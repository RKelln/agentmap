package parser

import (
	"reflect"
	"testing"
)

func TestComputeSections(t *testing.T) {
	tests := []struct {
		name       string
		headings   []Heading
		totalLines int
		want       []Section
	}{
		{
			name:       "empty headings list",
			headings:   nil,
			totalLines: 100,
			want:       nil,
		},
		{
			name:       "single heading spans entire file",
			headings:   []Heading{{Line: 1, Depth: 1, Text: "Title"}},
			totalLines: 50,
			want: []Section{
				{Heading: Heading{Line: 1, Depth: 1, Text: "Title"}, Start: 1, End: 50},
			},
		},
		{
			name: "multiple headings at same level",
			headings: []Heading{
				{Line: 1, Depth: 2, Text: "First"},
				{Line: 10, Depth: 2, Text: "Second"},
				{Line: 20, Depth: 2, Text: "Third"},
			},
			totalLines: 30,
			want: []Section{
				{Heading: Heading{Line: 1, Depth: 2, Text: "First"}, Start: 1, End: 9},
				{Heading: Heading{Line: 10, Depth: 2, Text: "Second"}, Start: 10, End: 19},
				{Heading: Heading{Line: 20, Depth: 2, Text: "Third"}, Start: 20, End: 30},
			},
		},
		{
			name: "nested h1 with h2 children",
			headings: []Heading{
				{Line: 1, Depth: 1, Text: "Main"},
				{Line: 5, Depth: 2, Text: "Sub A"},
				{Line: 15, Depth: 2, Text: "Sub B"},
			},
			totalLines: 25,
			want: []Section{
				{Heading: Heading{Line: 1, Depth: 1, Text: "Main"}, Start: 1, End: 25},
				{Heading: Heading{Line: 5, Depth: 2, Text: "Sub A"}, Start: 5, End: 14},
				{Heading: Heading{Line: 15, Depth: 2, Text: "Sub B"}, Start: 15, End: 25},
			},
		},
		{
			name: "three levels h1 h2 h3",
			headings: []Heading{
				{Line: 1, Depth: 1, Text: "Doc"},
				{Line: 5, Depth: 2, Text: "Section"},
				{Line: 10, Depth: 3, Text: "Subsection"},
				{Line: 20, Depth: 2, Text: "Other Section"},
			},
			totalLines: 30,
			want: []Section{
				{Heading: Heading{Line: 1, Depth: 1, Text: "Doc"}, Start: 1, End: 30},
				{Heading: Heading{Line: 5, Depth: 2, Text: "Section"}, Start: 5, End: 19},
				{Heading: Heading{Line: 10, Depth: 3, Text: "Subsection"}, Start: 10, End: 19},
				{Heading: Heading{Line: 20, Depth: 2, Text: "Other Section"}, Start: 20, End: 30},
			},
		},
		{
			name: "consecutive headings no content between",
			headings: []Heading{
				{Line: 1, Depth: 1, Text: "A"},
				{Line: 2, Depth: 2, Text: "B"},
				{Line: 3, Depth: 2, Text: "C"},
			},
			totalLines: 10,
			want: []Section{
				{Heading: Heading{Line: 1, Depth: 1, Text: "A"}, Start: 1, End: 10},
				{Heading: Heading{Line: 2, Depth: 2, Text: "B"}, Start: 2, End: 2},
				{Heading: Heading{Line: 3, Depth: 2, Text: "C"}, Start: 3, End: 10},
			},
		},
		{
			name: "headings at end of file",
			headings: []Heading{
				{Line: 1, Depth: 1, Text: "Title"},
				{Line: 48, Depth: 2, Text: "Last"},
			},
			totalLines: 50,
			want: []Section{
				{Heading: Heading{Line: 1, Depth: 1, Text: "Title"}, Start: 1, End: 50},
				{Heading: Heading{Line: 48, Depth: 2, Text: "Last"}, Start: 48, End: 50},
			},
		},
		{
			name: "h3 does not end h2 section",
			headings: []Heading{
				{Line: 1, Depth: 2, Text: "Parent"},
				{Line: 10, Depth: 3, Text: "Child"},
			},
			totalLines: 20,
			want: []Section{
				{Heading: Heading{Line: 1, Depth: 2, Text: "Parent"}, Start: 1, End: 20},
				{Heading: Heading{Line: 10, Depth: 3, Text: "Child"}, Start: 10, End: 20},
			},
		},
		{
			name: "h2 ends h1 but h3 does not",
			headings: []Heading{
				{Line: 1, Depth: 1, Text: "Root"},
				{Line: 5, Depth: 3, Text: "Deep"},
				{Line: 15, Depth: 2, Text: "Mid"},
			},
			totalLines: 25,
			want: []Section{
				{Heading: Heading{Line: 1, Depth: 1, Text: "Root"}, Start: 1, End: 25},
				{Heading: Heading{Line: 5, Depth: 3, Text: "Deep"}, Start: 5, End: 14},
				{Heading: Heading{Line: 15, Depth: 2, Text: "Mid"}, Start: 15, End: 25},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ComputeSections(tt.headings, tt.totalLines)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ComputeSections() = %v, want %v", got, tt.want)
			}
		})
	}
}
