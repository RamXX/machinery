defmodule Machinery.Assurance.JSON do
  @moduledoc false

  # Bounded JSON encoding of the closed reporter vocabulary. Strings are
  # escaped byte-wise (quote, backslash and C0 controls); binaries are cut
  # to the caller bound before escaping; everything else is one of the
  # closed literal shapes of the reporter.

  def escape(bin, bound) when is_binary(bin) do
    bin |> cut(bound) |> do_escape(<<>>)
  end

  defp cut(bin, bound) when byte_size(bin) <= bound, do: bin
  defp cut(bin, bound), do: binary_part(bin, 0, bound)

  defp do_escape(<<>>, acc), do: acc

  defp do_escape(<<byte, rest::binary>>, acc) do
    part =
      case byte do
        ?" -> "\\\""
        ?\\ -> "\\\\"
        ?\n -> "\\n"
        ?\r -> "\\r"
        ?\t -> "\\t"
        c when c < 0x20 -> "\\u" <> pad(Integer.to_string(c, 16), 4)
        c -> <<c>>
      end

    do_escape(rest, <<acc::binary, part::binary>>)
  end

  defp pad(hex, width) do
    case String.length(hex) do
      len when len >= width -> hex
      len -> String.duplicate("0", width - len) <> hex
    end
  end

  def int(value) when is_integer(value), do: Integer.to_string(value)
  def bool(value) when is_boolean(value), do: to_string(value)
  def atom(value) when is_atom(value), do: "\"" <> escape(to_string(value), 4096) <> "\""
  def string(value) when is_binary(value), do: "\"" <> escape(value, 4096) <> "\""

  def kv(pairs) do
    inner =
      pairs
      |> Enum.map(fn {key, encoded} -> "\"" <> escape(to_string(key), 256) <> "\":" <> encoded end)
      |> Enum.join(",")

    "{" <> inner <> "}"
  end

  def line(type, payload) do
    "{\"t\":" <> string(type) <> ",\"d\":" <> payload <> "}\n"
  end
end
