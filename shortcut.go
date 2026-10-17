package main

import "time"

type pressSequence struct {
	first    time.Time
	cooldown time.Time
	count    int
}

func (p *pressSequence) add(now time.Time) bool {
	if now.Before(p.cooldown) {
		return false
	}
	if p.count == 0 || now.Sub(p.first) > 2*time.Second {
		p.first = now
		p.count = 0
	}
	p.count++
	if p.count < 3 {
		return false
	}
	p.count = 0
	p.cooldown = now.Add(4 * time.Second)
	return true
}
