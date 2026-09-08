package util

import (
	"strings"
	"testing"
)

func TestCleanHTML(t *testing.T) {
	tests := []struct {
		input       string
		mustContain []string
		mustOmit    []string
	}{
		{
			input:       `<p>Hello <strong>world</strong></p>`,
			mustContain: []string{`<p>Hello <strong>world</strong></p>`},
			mustOmit:    nil,
		},
		{
			input:       `<div><script>alert('xss')</script><p>Safe content</p></div>`,
			mustContain: []string{`<div><p>Safe content</p></div>`},
			mustOmit:    []string{`script`, `alert`},
		},
		{
			input:       `<a href="javascript:alert(1)" onclick="steal()">Link</a>`,
			mustContain: []string{`<a>Link</a>`},
			mustOmit:    []string{`javascript`, `onclick`, `steal`},
		},
		{
			input:       `<img src="https://img.com/a.jpg" onerror="evil()" alt="pic" />`,
			mustContain: []string{`src="https://img.com/a.jpg"`, `alt="pic"`},
			mustOmit:    []string{`onerror`, `evil`},
		},
		{
			input:       `<table class="content-table"><tr><td colspan="2">Cell</td></tr></table>`,
			mustContain: []string{`<table class="content-table">`, `<td colspan="2">Cell</td>`},
			mustOmit:    nil,
		},
		{
			input:       `<p style="color: red; background-color: blue;">Styled</p>`,
			mustContain: []string{`style="color: red; background-color: blue;"`},
			mustOmit:    nil,
		},
		{
			input:       `<p style="width: expression(alert(1))">Bad style</p>`,
			mustContain: []string{`<p>Bad style</p>`},
			mustOmit:    []string{`expression`},
		},
	}

	for _, tc := range tests {
		res := CleanHTML(tc.input)
		for _, c := range tc.mustContain {
			if !strings.Contains(res, c) {
				t.Errorf("input %q result %q does not contain %q", tc.input, res, c)
			}
		}
		for _, o := range tc.mustOmit {
			if strings.Contains(res, o) {
				t.Errorf("input %q result %q contains disallowed %q", tc.input, res, o)
			}
		}
	}
}
