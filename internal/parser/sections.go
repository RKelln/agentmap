package parser

// Section represents a markdown heading with its computed line range.
type Section struct {
	Heading
	Start int // 1-indexed start line (same as Heading.Line)
	End   int // 1-indexed end line (inclusive)
}

// Len returns the number of lines in this section (inclusive).
func (s Section) Len() int {
	return s.End - s.Start + 1
}

// ComputeSections computes start/end line ranges for each heading.
// A section ends at the line before the next heading at the same or higher level,
// or at totalLines if no such heading exists.
func ComputeSections(headings []Heading, totalLines int) []Section {
	if len(headings) == 0 {
		return nil
	}

	sections := make([]Section, len(headings))

	for i, h := range headings {
		end := totalLines

		for j := i + 1; j < len(headings); j++ {
			if headings[j].Depth <= h.Depth {
				end = headings[j].Line - 1
				break
			}
		}

		sections[i] = Section{
			Heading: h,
			Start:   h.Line,
			End:     end,
		}
	}

	return sections
}
