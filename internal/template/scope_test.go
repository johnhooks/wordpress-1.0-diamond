package template

import "testing"

func TestPushData(t *testing.T) {
	type Post struct {
		Title   string `view:"the_title"`
		Content string `view:"the_content,raw"`
		ID      int64  `view:"id"`
	}

	root := NewScope(nil)
	root.Set("blog_name", "My Blog")

	post := Post{Title: "Hello", Content: "<p>World</p>", ID: 42}
	child := root.PushData(post)

	// Flattened fields resolve in child scope.
	val, ok := child.Lookup("the_title")
	if !ok || val != "Hello" {
		t.Errorf("the_title = %v, %v; want Hello, true", val, ok)
	}

	// Raw fields are wrapped in RawHTML.
	val, ok = child.Lookup("the_content")
	if !ok {
		t.Fatal("the_content not found")
	}
	if raw, ok := val.(RawHTML); !ok || string(raw) != "<p>World</p>" {
		t.Errorf("the_content = %v (%T); want RawHTML(<p>World</p>)", val, val)
	}

	// Non-raw fields are plain values.
	val, ok = child.Lookup("id")
	if !ok || val != int64(42) {
		t.Errorf("id = %v, %v; want 42, true", val, ok)
	}

	// Parent scope variables still resolve.
	val, ok = child.Lookup("blog_name")
	if !ok || val != "My Blog" {
		t.Errorf("blog_name = %v, %v; want My Blog, true", val, ok)
	}
}

func TestPushDataNil(t *testing.T) {
	root := NewScope(nil)
	root.Set("x", 1)

	child := root.PushData(nil)

	// Parent variable still accessible.
	val, ok := child.Lookup("x")
	if !ok || val != 1 {
		t.Errorf("x = %v, %v; want 1, true", val, ok)
	}
}

func TestPushDataEmbeddedStruct(t *testing.T) {
	type Base struct {
		BlogName string `view:"blog_name"`
	}
	type Page struct {
		Base
		Title string `view:"title"`
	}

	root := NewScope(nil)
	page := Page{Base: Base{BlogName: "Test"}, Title: "About"}
	child := root.PushData(page)

	val, ok := child.Lookup("blog_name")
	if !ok || val != "Test" {
		t.Errorf("blog_name = %v, %v; want Test, true", val, ok)
	}
	val, ok = child.Lookup("title")
	if !ok || val != "About" {
		t.Errorf("title = %v, %v; want About, true", val, ok)
	}
}
