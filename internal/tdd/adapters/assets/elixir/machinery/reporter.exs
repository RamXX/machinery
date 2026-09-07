defmodule Machinery.Assurance.Reporter do
  @moduledoc false

  # The embedded Machinery reporter of the elixir-exunit/v1 adapter: a named
  # GenServer that is both the witness collector (synchronous calls from the
  # machinery-check/v1 transport running inside the native test processes)
  # and the sole ExUnit formatter (lifecycle casts from ExUnit.Runner).
  # Every event is appended as one bounded JSON line to the private events
  # file named by MACHINERY_ASSURANCE_EVENTS, terminated by the closed
  # sentinel machinery:reporter:end after suite_finished; its absence proves
  # a truncated stream. File identities are relativized against the harness
  # working directory; absolute leftovers fail reconciliation as-is.

  alias Machinery.Assurance.JSON

  @sentinel "machinery:reporter:end"
  @message_bound 1024

  def init(_opts) do
    path = System.get_env("MACHINERY_ASSURANCE_EVENTS")

    if is_binary(path) and path != "" do
      case File.open(path, [:write]) do
        {:ok, device} ->
          case :erlang.register(Machinery.Assurance.Reporter, self()) do
            true ->
              {:ok,
               %{
                 device: device,
                 counters: %{tests: 0, failures: 0, skipped: 0, invalid: 0, excluded: 0}
               }}

            _ ->
              File.close(device)
              {:stop, :already_registered}
          end

        error ->
          {:stop, error}
      end
    else
      {:stop, :missing_events_path}
    end
  end

  # machinery-check/v1 witness transport (synchronous, from test processes).
  # A witness whose framework-recorded test pid is not the calling process,
  # or whose identity fields are not the closed shapes, is recorded with its
  # verdict and never trusted by reconciliation.

  def handle_call(
        {:witness, module, describe, test, id, value, file, line, verdict},
        _from,
        state
      )
      when is_atom(module) and is_atom(test) and is_binary(id) and is_boolean(value) and
             is_binary(file) and is_integer(line) and is_binary(verdict) do
    write(state, "witness", witness_payload(module, describe, test, id, value, file, line, verdict))
    {:reply, :ok, state}
  end

  def handle_call({:witness, _module, _describe, _test, _id, _value, _file, _line, verdict}, _from, state)
      when is_binary(verdict) do
    write(state, "witness", JSON.kv(verdict: JSON.string(verdict)))
    {:reply, :ok, state}
  end

  def handle_call(_other, _from, state) do
    write(state, "witness", JSON.kv(verdict: JSON.string("malformed")))
    {:reply, :ok, state}
  end

  # ExUnit formatter lifecycle (casts from ExUnit.Runner / EventManager).

  def handle_cast({:suite_started, opts}, state) do
    write(state, "suite_started", JSON.kv(
      seed: JSON.int(keyword(opts, :seed, -1)),
      max_cases: JSON.int(keyword(opts, :max_cases, -1)),
      trace: JSON.bool(keyword(opts, :trace, true)),
      timeout: JSON.int(keyword(opts, :timeout, -1)),
      max_failures: keyword(opts, :max_failures, :unset) |> life(),
      include: keyword(opts, :include, :unset) |> filters(),
      exclude: keyword(opts, :exclude, :unset) |> filters(),
      formatters: keyword(opts, :formatters, :unset) |> formatters(),
      dry_run: JSON.bool(keyword(opts, :dry_run, true)),
      repeat_until_failure: JSON.int(keyword(opts, :repeat_until_failure, -1))
    ))

    {:noreply, state}
  end

  def handle_cast({:module_started, test_module}, state) do
    write(state, "module_started", JSON.kv(module: JSON.atom(test_module.name)))
    {:noreply, state}
  end

  def handle_cast({:module_finished, test_module}, state) do
    write(state, "module_finished", JSON.kv(
      module: JSON.atom(test_module.name),
      state: JSON.string(module_state(test_module.state))
    ))

    {:noreply, state}
  end

  def handle_cast({:test_started, test}, state) do
    write(state, "test_started", test_identity(test, async: JSON.bool(async?(test))))
    {:noreply, state}
  end

  def handle_cast({:test_finished, test}, state) do
    counters =
      state.counters
      |> Map.update!(:tests, &(&1 + 1))
      |> count_state(test.state)

    write(state, "test_finished", test_identity(test,
      state: JSON.string(test_state(test.state)),
      class: JSON.string(failure_class(test.state)),
      message: JSON.string(failure_message(test.state))
    ))

    {:noreply, %{state | counters: counters}}
  end

  def handle_cast({:suite_finished, _times_us}, state) do
    write(state, "suite_finished", JSON.kv(
      tests: JSON.int(state.counters.tests),
      failures: JSON.int(state.counters.failures),
      skipped: JSON.int(state.counters.skipped),
      invalid: JSON.int(state.counters.invalid),
      excluded: JSON.int(state.counters.excluded)
    ))

    {:stop, :normal, close(state)}
  end

  def handle_cast(:max_failures_reached, state) do
    write(state, "max_failures_reached", JSON.kv([]))
    {:noreply, state}
  end

  def handle_cast(_other, state), do: {:noreply, state}

  def terminate(_reason, state), do: close(state)

  defp close(state) do
    case Map.get(state, :device) do
      nil ->
        :ok

      device ->
        IO.write(device, JSON.line(@sentinel, "{}"))
        File.close(device)
    end

    %{state | device: nil}
  end

  defp write(state, type, payload) do
    case Map.get(state, :device) do
      nil -> :ok
      device -> IO.write(device, JSON.line(type, payload))
    end
  end

  defp witness_payload(module, describe, test, id, value, file, line, verdict) do
    JSON.kv(
      module: JSON.atom(module),
      describe: JSON.string(describe_of(describe)),
      test: JSON.atom(test),
      id: JSON.string(id),
      value: JSON.bool(value),
      file: JSON.string(relative(file)),
      line: JSON.int(line_of(line)),
      verdict: JSON.string(verdict)
    )
  end

  defp test_identity(test, extra) do
    JSON.kv(
      [
        module: JSON.atom(test.module),
        describe: JSON.string(describe_of(Map.get(test.tags, :describe))),
        test: JSON.atom(test.name),
        file: JSON.string(relative(Map.get(test.tags, :file, ""))),
        line: JSON.int(line_of(Map.get(test.tags, :line)))
      ] ++ extra
    )
  end

  defp describe_of(describe) when is_binary(describe), do: describe
  defp describe_of(_), do: ""

  defp line_of(line) when is_integer(line), do: line
  defp line_of(_), do: 0

  defp keyword(opts, key, default) do
    case List.keyfind(opts, key, 0) do
      {^key, value} -> value
      _ -> default
    end
  end

  defp life(:infinity), do: JSON.string("infinity")
  defp life(value) when is_integer(value), do: JSON.int(value)
  defp life(_), do: JSON.string("unset")

  defp filters(:unset), do: JSON.string("unset")
  defp filters([]), do: "[]"
  defp filters(entries), do: "[" <> Enum.map_join(entries, ",", &filter_entry/1) <> "]"

  defp filter_entry(entry) when is_atom(entry), do: JSON.atom(entry)
  defp filter_entry(entry) when is_tuple(entry), do: JSON.atom(elem(entry, 0))
  defp filter_entry(_), do: JSON.string("other")

  defp formatters(:unset), do: JSON.string("unset")
  defp formatters(entries), do: "[" <> Enum.map_join(entries, ",", &JSON.atom(&1)) <> "]"

  defp relative(file) when is_binary(file) do
    case File.cwd() do
      {:ok, cwd} -> relativize(file, cwd)
      _ -> file
    end
  end

  defp relativize(file, cwd) do
    case file do
      ^cwd <> "/" <> rest -> rest
      ^cwd -> "."
      _ -> file
    end
  end

  defp async?(test), do: Map.get(test.tags, :async, false) == true

  defp module_state({:invalid, _}), do: "invalid"
  defp module_state(nil), do: "finished"
  defp module_state(_), do: "unknown"

  defp test_state(nil), do: "passed"
  defp test_state({:failed, _}), do: "failed"
  defp test_state({:skipped, _}), do: "skipped"
  defp test_state({:excluded, _}), do: "excluded"
  defp test_state({:invalid, _}), do: "invalid"
  defp test_state(_), do: "unknown"

  defp failure_class({:failed, failures}) do
    case failures do
      [{_kind, %ExUnit.AssertionError{}, _stack} | _] -> "ExUnit.AssertionError"
      [{:EXIT, _pid, reason} | _] -> "exit:" <> inspect(reason)
      [{kind, reason, _stack} | _] -> exception_class(kind, reason)
      _ -> "unknown"
    end
  rescue
    _ -> "unknown"
  end

  defp failure_class(_), do: ""

  defp failure_message({:failed, failures}) do
    case failures do
      [{_kind, %ExUnit.AssertionError{message: message}, _stack} | _] when is_binary(message) ->
        String.slice(message, 0, @message_bound)

      [{kind, reason, _stack} | _] ->
        banner(kind, reason)

      _ ->
        ""
    end
  rescue
    _ -> ""
  end

  defp failure_message(_), do: ""

  defp banner(kind, reason) do
    try do
      kind |> Exception.format_banner(reason) |> String.slice(0, @message_bound)
    rescue
      _ -> ""
    end
  end

  defp exception_class(kind, reason) do
    case reason do
      %{} = struct ->
        case struct.__struct__ do
          module when is_atom(module) -> module |> to_string() |> String.trim_leading("Elixir.")
          _ -> safe_inspect(kind)
        end

      _ ->
        safe_inspect(kind)
    end
  rescue
    _ -> safe_inspect(kind)
  end

  defp safe_inspect(value) do
    try do
      value |> inspect() |> String.slice(0, 64)
    rescue
      _ -> "unknown"
    end
  end

  defp count_state(counters, :passed), do: counters
  defp count_state(counters, {:failed, _}), do: Map.update!(counters, :failures, &(&1 + 1))
  defp count_state(counters, {:skipped, _}), do: Map.update!(counters, :skipped, &(&1 + 1))
  defp count_state(counters, {:excluded, _}), do: Map.update!(counters, :excluded, &(&1 + 1))
  defp count_state(counters, {:invalid, _}), do: Map.update!(counters, :invalid, &(&1 + 1))
  defp count_state(counters, _), do: counters
end
