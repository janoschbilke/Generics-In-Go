package main

import "myproject/collections"

func main() {
	s := collections.Stack[int]{}  // explizite Typ-Instanziierung
	s.Push(42)                     // inferierte Methoden-Instanziierung
	val := s.Pop()                 // inferierte Methoden-Instanziierung
	_ = val
}