defmodule Machinery.ConformanceTest do
  @moduledoc false
  use ExUnit.Case, async: true

  require Machinery.Check

  test "conformance witness executes native assertion", ctx do
    Machinery.Check.check(ctx, "conformance/witness", 6 * 7 == 42)
  end

  describe "conformance parent identity" do
    test "conformance nested identity", ctx do
      Machinery.Check.check(ctx, "conformance/nested", 2 * 7 == 14)
    end
  end

  test "conformance multiple assertions", ctx do
    Machinery.Check.check(ctx, "conformance/multi-a", 3 * 5 == 15)
    Machinery.Check.check(ctx, "conformance/multi-b", 4 * 5 == 20)
  end
end
