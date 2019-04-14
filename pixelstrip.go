package main

import (
	"encoding/json"
	"net/url"
	"time"
)

// PixelStrip -
type PixelStrip struct {
	client *PixelStripClient
}

// NewPixelStrip -
func NewPixelStrip(u *url.URL) *PixelStrip {
	return &PixelStrip{
		client: &PixelStripClient{
			url: u,
		},
	}
}

// Fill -
func (p *PixelStrip) Fill(c pixel) error {
	err := p.client.Send("fill", map[string]string{"colour": c.Hex()})
	return err
}

// Raw -
func (p *PixelStrip) Raw(px []pixel) error {
	j, err := json.Marshal(px)
	if err != nil {
		return err
	}

	err = p.client.SendBody("raw", nil, j)
	return err
}

// Clear -
func (p *PixelStrip) Clear() error {
	return p.client.Send("clear", nil)
}

// FadeOut -
func (p *PixelStrip) FadeOut(c pixel, d time.Duration) error {
	return p.Fade(c, mustParseHex("#000000"), d)
}

// Fade -
func (p *PixelStrip) Fade(from pixel, to pixel, d time.Duration) error {
	steps := 4.0
	t := d / time.Duration(steps)

	// use a maximum step time of 200ms
	if t > 100*time.Millisecond {
		t = 100 * time.Millisecond
		steps = float64(d / t)
	}

	fadeStart := time.Now()
	for i := 1.0; i <= steps; i++ {
		end := time.Now().Add(t)
		step := i / steps
		np := pixel{from.BlendHcl(to.Color, step)}
		// log.Printf("%v / %v == %v        t%s", i, steps, step, np.Color.Hex())
		err := p.Fill(np)
		if err != nil {
			return err
		}
		left := time.Until(end)
		time.Sleep(left)
	}
	fadeEnd := time.Now()
	fadeHist.Observe(fadeEnd.Sub(fadeStart).Seconds())
	fadeSumm.Observe(fadeEnd.Sub(fadeStart).Seconds())
	return nil
}
