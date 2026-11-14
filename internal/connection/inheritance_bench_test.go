package connection

import "testing"

func BenchmarkEffective_Depth5(b *testing.B) {
	root, _ := NewRootInfo()
	parent := root
	for i := 0; i < 5; i++ {
		c, _ := NewContainerInfo()
		c.Base().Inheritance.SetAll(true)
		_ = parent.AddChild(c)
		parent = c
	}
	leaf, _ := NewConnectionInfo()
	leaf.Inheritance.SetAll(true)
	_ = parent.AddChild(leaf)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = leaf.Effective()
	}
}
