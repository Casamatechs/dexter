defmodule AppWithStyler.MixProject do
  use Mix.Project

  def project do
    [
      app: :app_with_styler,
      version: "0.1.0",
      elixir: "~> 1.18",
      deps: deps()
    ]
  end

  defp deps do
    [
      {:styler, "~> 1.4.2", only: [:dev, :test], runtime: false, git: "https://github.com/remoteoss/elixir-styler"}
    ]
  end
end
