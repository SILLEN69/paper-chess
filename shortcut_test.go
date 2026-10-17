package main

import (
	"testing"
	"time"
)

func TestPowerSequence(t *testing.T) {
	p := pressSequence{}
	start := time.Now()
	if p.add(start) || p.add(start.Add(400*time.Millisecond)) {
		t.Fatal("early trigger")
	}
	if !p.add(start.Add(800 * time.Millisecond)) {
		t.Fatal("no triple trigger")
	}
	if p.add(start.Add(time.Second)) {
		t.Fatal("cooldown ignored")
	}
	if p.add(start.Add(6*time.Second)) || p.add(start.Add(9*time.Second)) || p.add(start.Add(9500*time.Millisecond)) {
		t.Fatal("old press was counted")
	}
	if !p.add(start.Add(10 * time.Second)) {
		t.Fatal("fresh triple did not trigger")
	}
}
