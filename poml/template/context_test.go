package template

import "testing"

func TestContext(t *testing.T) {
	ctx := NewContext(map[string]any{
		"name": "Alice",
		"age":  30,
	})

	t.Run("Get simple", func(t *testing.T) {
		val, ok := ctx.Get("name")
		if !ok || val != "Alice" {
			t.Errorf("Get(name) = %v, %v", val, ok)
		}
	})

	t.Run("Get missing", func(t *testing.T) {
		_, ok := ctx.Get("missing")
		if ok {
			t.Errorf("Get(missing) should return false")
		}
	})

	t.Run("Set", func(t *testing.T) {
		ctx.Set("city", "NYC")
		val, ok := ctx.Get("city")
		if !ok || val != "NYC" {
			t.Errorf("Set/Get failed")
		}
	})

	t.Run("Child context", func(t *testing.T) {
		child := ctx.Child()
		child.Set("local", "value")

		// Child has its own vars
		val, ok := child.Get("local")
		if !ok || val != "value" {
			t.Errorf("child.Get(local) failed")
		}

		// Child inherits parent vars
		val, ok = child.Get("name")
		if !ok || val != "Alice" {
			t.Errorf("child.Get(name) should inherit from parent")
		}

		// Parent doesn't see child vars
		_, ok = ctx.Get("local")
		if ok {
			t.Errorf("parent should not see child vars")
		}
	})

	t.Run("Nested access", func(t *testing.T) {
		ctx.Set("user", map[string]any{
			"name": "Bob",
			"address": map[string]any{
				"city": "LA",
			},
		})

		val, ok := ctx.Get("user.name")
		if !ok || val != "Bob" {
			t.Errorf("Get(user.name) = %v, %v", val, ok)
		}

		val, ok = ctx.Get("user.address.city")
		if !ok || val != "LA" {
			t.Errorf("Get(user.address.city) = %v, %v", val, ok)
		}
	})
}

func TestContextAll(t *testing.T) {
	parent := NewContext(map[string]any{"a": 1})
	child := parent.Child()
	child.Set("b", 2)

	all := child.All()
	if all["a"] != 1 || all["b"] != 2 {
		t.Errorf("All() = %v", all)
	}
}

func TestLoopContext(t *testing.T) {
	lc := NewLoopContext(0, 3, "first")
	if !lc.First {
		t.Error("First should be true for index 0")
	}
	if lc.Last {
		t.Error("Last should be false for index 0")
	}

	lc = NewLoopContext(2, 3, "last")
	if lc.First {
		t.Error("First should be false for last index")
	}
	if !lc.Last {
		t.Error("Last should be true for last index")
	}
}
