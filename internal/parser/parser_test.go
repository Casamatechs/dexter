package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTempFile(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.ex")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestParseFile_SingleModule(t *testing.T) {
	path := writeTempFile(t, `defmodule MyApp.Handlers.Foo do
  def bar(arg) do
    :ok
  end

  defp baz do
    :secret
  end
end
`)

	defs, err := ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if len(defs) != 3 {
		t.Fatalf("expected 3 definitions, got %d", len(defs))
	}

	// Module
	if defs[0].Module != "MyApp.Handlers.Foo" || defs[0].Kind != "module" || defs[0].Line != 1 {
		t.Errorf("unexpected module def: %+v", defs[0])
	}

	// Public function
	if defs[1].Module != "MyApp.Handlers.Foo" || defs[1].Function != "bar" || defs[1].Kind != "def" || defs[1].Line != 2 {
		t.Errorf("unexpected def: %+v", defs[1])
	}

	// Private function
	if defs[2].Module != "MyApp.Handlers.Foo" || defs[2].Function != "baz" || defs[2].Kind != "defp" || defs[2].Line != 6 {
		t.Errorf("unexpected defp: %+v", defs[2])
	}
}

func TestParseFile_MultipleFunctionHeads(t *testing.T) {
	path := writeTempFile(t, `defmodule MyApp.Webhooks do
  def process_event("completed", payload) do
    :ok
  end

  def process_event("declined", payload) do
    :declined
  end

  def process_event(_, _) do
    :unknown
  end
end
`)

	defs, err := ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}

	funcDefs := 0
	for _, d := range defs {
		if d.Function == "process_event" {
			funcDefs++
			if d.Module != "MyApp.Webhooks" || d.Kind != "def" {
				t.Errorf("unexpected process_event def: %+v", d)
			}
		}
	}
	if funcDefs != 3 {
		t.Errorf("expected 3 process_event heads, got %d", funcDefs)
	}
}

func TestParseFile_NestedModules(t *testing.T) {
	path := writeTempFile(t, `defmodule MyApp.Outer do
  def outer_func do
    :ok
  end

  defmodule MyApp.Outer.Inner do
    def inner_func do
      :ok
    end
  end
end
`)

	defs, err := ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}

	modules := map[string]bool{}
	for _, d := range defs {
		if d.Kind == "module" {
			modules[d.Module] = true
		}
	}

	if !modules["MyApp.Outer"] {
		t.Error("missing MyApp.Outer module")
	}
	if !modules["MyApp.Outer.Inner"] {
		t.Error("missing MyApp.Outer.Inner module")
	}

	// inner_func should belong to MyApp.Outer.Inner
	for _, d := range defs {
		if d.Function == "inner_func" && d.Module != "MyApp.Outer.Inner" {
			t.Errorf("inner_func should belong to MyApp.Outer.Inner, got %s", d.Module)
		}
	}
}

func TestParseFile_Macros(t *testing.T) {
	path := writeTempFile(t, `defmodule MyApp.Macros do
  defmacro my_macro(arg) do
    quote do: unquote(arg)
  end

  defmacrop private_macro do
    :ok
  end
end
`)

	defs, err := ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}

	kinds := map[string]string{}
	for _, d := range defs {
		if d.Function != "" {
			kinds[d.Function] = d.Kind
		}
	}

	if kinds["my_macro"] != "defmacro" {
		t.Errorf("expected defmacro for my_macro, got %s", kinds["my_macro"])
	}
	if kinds["private_macro"] != "defmacrop" {
		t.Errorf("expected defmacrop for private_macro, got %s", kinds["private_macro"])
	}
}

func TestParseFile_FunctionWithQuestionMark(t *testing.T) {
	path := writeTempFile(t, `defmodule MyApp.Guards do
  def valid?(thing) do
    true
  end

  def process!(thing) do
    :ok
  end
end
`)

	defs, err := ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}

	funcs := map[string]bool{}
	for _, d := range defs {
		if d.Function != "" {
			funcs[d.Function] = true
		}
	}

	if !funcs["valid?"] {
		t.Error("missing valid? function")
	}
	if !funcs["process!"] {
		t.Error("missing process! function")
	}
}

func TestParseFile_HeredocDefmoduleIgnored(t *testing.T) {
	path := writeTempFile(t, `defmodule Tesla do
  @moduledoc """
  Example:

      defmodule MyApi do
        def new(opts) do
          Tesla.client(middleware, adapter)
        end
      end
  """

  def client(middleware, adapter \\ nil), do: build(middleware, adapter)

  defp build(middleware, adapter) do
    :ok
  end
end
`)

	defs, err := ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}

	// Should only have Tesla module, not MyApi
	modules := map[string]bool{}
	for _, d := range defs {
		if d.Kind == "module" {
			modules[d.Module] = true
		}
	}
	if !modules["Tesla"] {
		t.Error("missing Tesla module")
	}
	if modules["MyApi"] {
		t.Error("MyApi from heredoc should not be indexed")
	}

	// client should belong to Tesla
	found := false
	for _, d := range defs {
		if d.Function == "client" {
			found = true
			if d.Module != "Tesla" {
				t.Errorf("client should belong to Tesla, got %s", d.Module)
			}
		}
	}
	if !found {
		t.Error("missing client function")
	}

	// build should belong to Tesla too
	for _, d := range defs {
		if d.Function == "build" && d.Module != "Tesla" {
			t.Errorf("build should belong to Tesla, got %s", d.Module)
		}
	}
}

func TestParseFile_SigillHeredocIgnored(t *testing.T) {
	path := writeTempFile(t, `defmodule MyApp.Docs do
  @doc ~S"""
  Usage:

      defmodule Example do
        def example_func do
          :ok
        end
      end
  """
  def real_func do
    :ok
  end
end
`)

	defs, err := ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}

	modules := map[string]bool{}
	funcs := map[string]string{}
	for _, d := range defs {
		if d.Kind == "module" {
			modules[d.Module] = true
		}
		if d.Function != "" {
			funcs[d.Function] = d.Module
		}
	}

	if modules["Example"] {
		t.Error("Example from sigil heredoc should not be indexed")
	}
	if funcs["example_func"] != "" {
		t.Error("example_func from sigil heredoc should not be indexed")
	}
	if funcs["real_func"] != "MyApp.Docs" {
		t.Errorf("real_func should belong to MyApp.Docs, got %s", funcs["real_func"])
	}
}

func TestParseFile_ModuleNestingRestoresAfterEnd(t *testing.T) {
	// After an inner module's `end`, functions should belong to the outer module
	path := writeTempFile(t, `defmodule MyApp.Outer do
  defmodule MyApp.Outer.Inner do
    def inner_func do
      :ok
    end
  end

  def outer_func do
    :ok
  end
end
`)

	defs, err := ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}

	funcs := map[string]string{}
	for _, d := range defs {
		if d.Function != "" {
			funcs[d.Function] = d.Module
		}
	}

	if funcs["inner_func"] != "MyApp.Outer.Inner" {
		t.Errorf("inner_func should belong to MyApp.Outer.Inner, got %s", funcs["inner_func"])
	}
	if funcs["outer_func"] != "MyApp.Outer" {
		t.Errorf("outer_func should belong to MyApp.Outer, got %s", funcs["outer_func"])
	}
}

func TestParseFile_SingleLineDefWithDefaultArg(t *testing.T) {
	path := writeTempFile(t, `defmodule Tesla do
  def client(middleware, adapter \\ nil), do: build(middleware, adapter)
end
`)

	defs, err := ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}

	found := false
	for _, d := range defs {
		if d.Function == "client" {
			found = true
			if d.Module != "Tesla" {
				t.Errorf("client should belong to Tesla, got %s", d.Module)
			}
			if d.Line != 2 {
				t.Errorf("client should be on line 2, got %d", d.Line)
			}
		}
	}
	if !found {
		t.Error("missing client function")
	}
}

func TestParseFile_InlineModuledoc(t *testing.T) {
	path := writeTempFile(t, `defmodule MyApp.Simple do
  @moduledoc "A simple module"

  def hello do
    :ok
  end
end
`)

	defs, err := ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}

	found := false
	for _, d := range defs {
		if d.Function == "hello" && d.Module == "MyApp.Simple" {
			found = true
		}
	}
	if !found {
		t.Error("missing hello function in MyApp.Simple")
	}
}

