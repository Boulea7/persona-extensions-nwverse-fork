package pluginkit

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"testing"
)

func TestGroupHandlersRegister(t *testing.T) {
	var got []string
	g := GroupHandlers{
		Tool: func(_ context.Context, c GroupToolCall) (any, error) {
			got = append(got, "tool "+c.Thread+" "+c.Name+" "+string(c.Args))
			return map[string]any{"messages": []any{}}, nil
		},
		Fallback: func(_ context.Context, f GroupFallback) (GroupFallbackResult, error) {
			got = append(got, "fallback "+f.Batch+" "+f.Events[0].ID)
			return GroupFallbackResult{New: []GroupNew{{Key: "k", Messages: []string{"m1"}, RuleTier: "fyi", Title: "t"}}}, nil
		},
	}
	h := map[string]Handler{}
	g.Register(h, "src")
	if _, ok := h[MGroupAppend]; ok || h[MGroupTool] == nil || h[MGroupFallback] == nil {
		t.Fatalf("registered %v", keys(h))
	}
	ctx := context.Background()
	if _, err := h[MGroupTool](ctx, nil, json.RawMessage(`{"contribution":"src","thread":"c1","name":"search","args":{"query":"x"}}`)); err != nil {
		t.Fatal(err)
	}
	res, err := h[MGroupFallback](ctx, nil, json.RawMessage(`{"thread":"c1","batch":"c1@2","events":[{"id":"e1","meta":null,"done":false}]}`))
	if err != nil || res.(GroupFallbackResult).New[0].Key != "k" {
		t.Fatalf("fallback = %+v %v", res, err)
	}
	if !slices.Equal(got, []string{`tool c1 search {"query":"x"}`, "fallback c1@2 e1"}) {
		t.Fatalf("calls %q", got)
	}
	var pe *Error
	if _, err := h[MGroupTool](ctx, nil, json.RawMessage(`{"contribution":"other"}`)); !errors.As(err, &pe) || pe.Code != CodeNoContribution {
		t.Fatalf("another contribution: %v", err)
	}
	if caps := g.Capabilities(); !slices.Equal(caps, []string{CapGroup}) {
		t.Fatalf("without Append: %v", caps)
	}
	g.Append = func(_ context.Context, a GroupAppend) (any, error) {
		return map[string]any{"id": "e1", "meta": a.Meta}, nil
	}
	g.Register(h, "src")
	if res, err := h[MGroupAppend](ctx, nil, json.RawMessage(`{"thread":"c1","event":"","meta":null,"messages":["m1"]}`)); err != nil || res == nil {
		t.Fatalf("append = %v %v", res, err)
	}
	if caps := g.Capabilities(); !slices.Equal(caps, []string{CapGroup, CapMeta}) {
		t.Fatalf("with Append: %v", caps)
	}
}

func keys(h map[string]Handler) []string {
	var out []string
	for k := range h {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}

func groupManifest(caps []string, desc *string) []byte {
	c := map[string]any{"id": "src", "capabilities": caps}
	if desc != nil {
		c["meta_description"] = *desc
	}
	raw, _ := json.Marshal(map[string]any{"contributes": map[string]any{"connectors": []any{c}}})
	return raw
}

func TestCheckGroupManifest(t *testing.T) {
	desc := "meta 是……"
	empty := " "
	long := strings.Repeat("长", MetaDescriptionMost+1)
	tool := func(context.Context, GroupToolCall) (any, error) { return nil, nil }
	appendFn := func(context.Context, GroupAppend) (any, error) { return nil, nil }
	withAppend := GroupHandlers{Tool: tool, Append: appendFn}
	withoutAppend := GroupHandlers{Tool: tool}
	for name, c := range map[string]struct {
		caps []string
		desc *string
		g    GroupHandlers
		want string // "" passes
	}{
		"meta with append":             {[]string{"inbox", "group", "meta"}, &desc, withAppend, ""},
		"no meta without append":       {[]string{"inbox", "group"}, nil, withoutAppend, ""},
		"no events at all":             {[]string{"inbox"}, nil, GroupHandlers{}, ""},
		"append, meta not declared":    {[]string{"inbox", "group"}, nil, withAppend, "declare meta"},
		"meta, no append":              {[]string{"inbox", "group", "meta"}, &desc, withoutAppend, "without an Append handler"},
		"meta, no description":         {[]string{"inbox", "group", "meta"}, nil, withAppend, "needs meta_description"},
		"meta, a blank description":    {[]string{"inbox", "group", "meta"}, &empty, withAppend, "needs meta_description"},
		"a description too long":       {[]string{"inbox", "group", "meta"}, &long, withAppend, "longer than"},
		"a description without meta":   {[]string{"inbox", "group"}, &desc, withoutAppend, "without meta"},
		"meta without group":           {[]string{"inbox", "meta"}, &desc, GroupHandlers{Append: appendFn}, "meta without group"},
		"handlers, group not declared": {[]string{"inbox"}, nil, withoutAppend, "group not declared"},
		"group without an inbox":       {[]string{"group"}, nil, withoutAppend, "without inbox"},
	} {
		err := CheckGroupManifest(groupManifest(c.caps, c.desc), "src", c.g)
		switch {
		case c.want == "" && err != nil:
			t.Errorf("%s: %v", name, err)
		case c.want != "" && (err == nil || !strings.Contains(err.Error(), c.want)):
			t.Errorf("%s: %v, want %q", name, err, c.want)
		}
	}
	if err := CheckGroupManifest(groupManifest(nil, nil), "nope", GroupHandlers{}); err == nil {
		t.Fatal("a connector the manifest does not have")
	}
}
