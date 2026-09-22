defmodule PacketFixture.ReleaseFixture do
  @moduledoc false

  def release_record(id), do: %{id: id, state: :released}
end
