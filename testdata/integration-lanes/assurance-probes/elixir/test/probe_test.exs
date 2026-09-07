defmodule AssuranceProbeTest do
  use ExUnit.Case, async: false

  test "assurance runtime probe executes a native ExUnit closure" do
    assert 6 * 7 == 42
  end
end
