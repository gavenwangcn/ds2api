package prompt

import "testing"

func TestStringifyToolCallArgumentsPreservesConcatenatedJSON(t *testing.T) {
	got := StringifyToolCallArguments(`{}{"query":"测试工具调用"}`)
	if got != `{}{"query":"测试工具调用"}` {
		t.Fatalf("expected raw concatenated JSON to be preserved, got %q", got)
	}
}

func TestFormatToolCallsForPromptXML(t *testing.T) {
	got := FormatToolCallsForPrompt([]any{
		map[string]any{
			"id": "call_1",
			"function": map[string]any{
				"name":      "search_web",
				"arguments": map[string]any{"query": "latest"},
			},
		},
	})
	if got == "" {
		t.Fatal("expected non-empty formatted tool calls")
	}
	if got != "<tools>\n  <tool_call>\n    <tool_name>search_web</tool_name>\n    <param>\n      <query><![CDATA[latest]]></query>\n    </param>\n  </tool_call>\n</tools>" {
		t.Fatalf("unexpected formatted tool call XML: %q", got)
	}
}

func TestFormatToolCallsForPromptEscapesXMLEntities(t *testing.T) {
	got := FormatToolCallsForPrompt([]any{
		map[string]any{
			"name":      "search<&>",
			"arguments": `{"q":"a < b && c > d"}`,
		},
	})
	want := "<tools>\n  <tool_call>\n    <tool_name>search&lt;&amp;&gt;</tool_name>\n    <param>\n      <q><![CDATA[a < b && c > d]]></q>\n    </param>\n  </tool_call>\n</tools>"
	if got != want {
		t.Fatalf("unexpected escaped tool call XML: %q", got)
	}
}

func TestFormatToolCallsForPromptUsesCDATAForMultilineContent(t *testing.T) {
	got := FormatToolCallsForPrompt([]any{
		map[string]any{
			"name": "write_file",
			"arguments": map[string]any{
				"path":    "script.sh",
				"content": "#!/bin/bash\nprintf \"hello\"\n",
			},
		},
	})
	want := "<tools>\n  <tool_call>\n    <tool_name>write_file</tool_name>\n    <param>\n      <content><![CDATA[#!/bin/bash\nprintf \"hello\"\n]]></content>\n      <path><![CDATA[script.sh]]></path>\n    </param>\n  </tool_call>\n</tools>"
	if got != want {
		t.Fatalf("unexpected multiline cdata tool call XML: %q", got)
	}
}
