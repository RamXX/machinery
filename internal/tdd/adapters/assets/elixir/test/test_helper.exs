# machinery.elixir-exunit/v1 embedded bootstrap (byte-pinned by
# internal/tdd/adapters/elixir.go). The load order is fixed: the bounded
# JSON codec, the embedded Machinery reporter, the machinery-check/v1
# transport, then ExUnit.start with the closed formatter replacement. The
# effective ExUnit options (seed, max_cases, formatters, ...) are merged
# from the frozen mix argv after this file evaluates and are reported by
# the embedded reporter itself; nothing here is project-authorable.
Code.require_file("../machinery/json.exs", __DIR__)
Code.require_file("../machinery/reporter.exs", __DIR__)
Code.require_file("../machinery/check.exs", __DIR__)

ExUnit.start(formatters: [Machinery.Assurance.Reporter], max_failures: :infinity)
