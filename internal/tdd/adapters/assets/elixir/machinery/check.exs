defmodule Machinery.Check do
  @moduledoc false

  # machinery-check/v1 assertion transport of the elixir-exunit/v1 adapter
  # (byte-pinned by internal/tdd/adapters/elixir.go). check/3 is a macro so
  # the registered call site is the exact frozen test source line; the
  # strictly-boolean condition is evaluated exactly once at execution time
  # and every evaluation emits exactly one witness through the embedded
  # reporter before the native framework classifies the outcome.

  defmacro check(ctx, assertion_id, condition) do
    unless is_binary(assertion_id) do
      raise ArgumentError,
            "machinery-check/v1 assertion id must be a literal string at " <>
              to_string(__CALLER__.file) <> ":#{__CALLER__.line}"
    end

    file = __CALLER__.file
    line = __CALLER__.line

    quote do
      Machinery.Assurance.Witness.witness(
        unquote(ctx),
        unquote(assertion_id),
        unquote(condition),
        unquote(file),
        unquote(line)
      )
    end
  end
end

defmodule Machinery.Assurance.Witness do
  @moduledoc false

  # The runtime half of the transport. It runs inside the native test
  # process: the witness is recorded synchronously BEFORE the native
  # failure is raised, so a false condition can never be reported as a
  # passing evaluation. Test identity comes from the framework-owned
  # context (module, describe, test and the framework-recorded test pid);
  # a context whose test pid is not the calling process is reported as a
  # pid-mismatch and never trusted.

  def witness(ctx, id, condition, file, line)
      when is_map(ctx) and is_binary(id) and is_integer(line) do
    value =
      cond do
        is_boolean(condition) -> condition
        true -> raise ArgumentError, "machinery-check/v1 condition must be strictly boolean"
      end

    module = Map.get(ctx, :module)
    describe = Map.get(ctx, :describe)
    test = Map.get(ctx, :test)
    pid = Map.get(ctx, :test_pid)
    verdict = if pid === self(), do: "ok", else: "pid-mismatch"

    case GenServer.call(Machinery.Assurance.Reporter, {:witness, module, describe, test, id, value, file, line, verdict}) do
      :ok ->
        if value do
          :ok
        else
          raise ExUnit.AssertionError,
            message:
              "machinery-check/v1 assertion " <> id <> " evaluated false at " <>
                rel(file) <> ":" <> Integer.to_string(line)
        end

      other ->
        raise ArgumentError, "machinery-check/v1 witness transport failed: " <> inspect(other)
    end
  end

  def rel(file) when is_binary(file) do
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
end
