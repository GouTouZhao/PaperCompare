package pdfparser

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/ledongthuc/pdf"
)

type PageContent struct {
	PageNumber int    `json:"page_number"`
	RawText    string `json:"raw_text"`
	CleanText  string `json:"clean_text"`
}

type SectionContent struct {
	SectionTitle string `json:"section_title"`
	SectionType  string `json:"section_type"`
	StartPage    int    `json:"start_page"`
	EndPage      int    `json:"end_page"`
	Content      string `json:"content"`
	SectionOrder int    `json:"section_order"`
}

type ParsedDocument struct {
	PageCount int              `json:"page_count"`
	Title     string           `json:"title"`
	Abstract  string           `json:"abstract"`
	DOI       string           `json:"doi"`
	Pages     []PageContent    `json:"pages"`
	Sections  []SectionContent `json:"sections"`
}

var (
	doiRegex     = regexp.MustCompile(`(?i)\b(10\.\d{4,9}/[-._;()/:A-Z0-9]+)\b`)
	sectionRegex = regexp.MustCompile(`(?i)^\s*(?:(?:[0-9]{1,2}(?:\.[0-9]{1,2})*|[IVX]{1,4})[\.\s]+)?(abstract|introduction|background|related\s+work|preliminaries|method|methodology|proposed\s+method|proposed\s+framework|approach|system\s+model|architecture|experiments?|experimental\s+setup|evaluation|results?(?:\s+and\s+discussion)?|discussion|ablation\s+study|limitations?|conclusion(?:s|\s+and\s+future\s+work)?|future\s+work|references|bibliography)\s*$`)
)

func ParsePDFFile(filePath string) (*ParsedDocument, error) {
	f, r, err := pdf.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open pdf: %w", err)
	}
	defer f.Close()

	totalPage := r.NumPage()
	pages := make([]PageContent, 0, totalPage)

	for pageIndex := 1; pageIndex <= totalPage; pageIndex++ {
		p := r.Page(pageIndex)
		if p.V.IsNull() {
			pages = append(pages, PageContent{PageNumber: pageIndex})
			continue
		}
		rawText, err := extractPageText(p)
		if err != nil {
			rawText = ""
		}
		clean := cleanText(rawText)
		pages = append(pages, PageContent{
			PageNumber: pageIndex,
			RawText:    rawText,
			CleanText:  clean,
		})
	}

	doc := &ParsedDocument{
		PageCount: totalPage,
		Pages:     pages,
	}

	if len(pages) > 0 {
		firstPage := pages[0].CleanText
		doc.DOI = extractDOI(firstPage)
		doc.Title = guessTitle(firstPage)
		doc.Abstract = guessAbstract(firstPage)
	}
	doc.Sections = identifySections(pages)

	return doc, nil
}

func extractPageText(p pdf.Page) (string, error) {
	rows, err := p.GetTextByRow()
	if err == nil && len(rows) > 0 {
		var buf bytes.Buffer
		for _, row := range rows {
			var line strings.Builder
			for i, word := range row.Content {
				s := strings.TrimSpace(word.S)
				if s == "" {
					continue
				}
				if i > 0 {
					line.WriteByte(' ')
				}
				line.WriteString(s)
			}
			lineStr := strings.TrimSpace(line.String())
			if lineStr != "" {
				buf.WriteString(lineStr)
				buf.WriteByte('\n')
			}
		}
		if buf.Len() > 0 {
			return buf.String(), nil
		}
	}
	return p.GetPlainText(nil)
}

func cleanText(raw string) string {
	raw = strings.ToValidUTF8(raw, "")
	raw = strings.ReplaceAll(raw, "\x00", "")
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	lines := strings.Split(raw, "\n")
	var out []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		var sb strings.Builder
		prevSpace := false
		for _, r := range trimmed {
			if unicode.IsSpace(r) {
				if !prevSpace {
					sb.WriteByte(' ')
					prevSpace = true
				}
			} else if unicode.IsPrint(r) {
				sb.WriteRune(r)
				prevSpace = false
			}
		}
		cleanedLine := strings.TrimSpace(sb.String())
		if cleanedLine != "" {
			out = append(out, cleanedLine)
		}
	}
	return strings.Join(out, "\n")
}

func extractDOI(text string) string {
	m := doiRegex.FindStringSubmatch(text)
	if len(m) > 1 {
		return strings.TrimRight(m[1], ".,;)")
	}
	return ""
}

func guessTitle(firstPage string) string {
	lines := strings.Split(firstPage, "\n")
	for _, l := range lines {
		l = strings.TrimSpace(l)
		lower := strings.ToLower(l)
		if len(l) < 8 || len(l) > 220 {
			continue
		}
		if strings.HasPrefix(lower, "arxiv:") || strings.Contains(lower, "ieee") || strings.Contains(lower, "acm") || strings.Contains(lower, "preprint") || strings.Contains(lower, "proceedings") {
			continue
		}
		return l
	}
	return ""
}

func guessAbstract(firstPage string) string {
	lower := strings.ToLower(firstPage)
	idx := strings.Index(lower, "abstract")
	if idx == -1 {
		return ""
	}
	rest := firstPage[idx+len("abstract"):]
	rest = strings.TrimLeft(rest, " :.—-\n\t")
	endMarkers := []string{"\n1 introduction", "\n1. introduction", "\ni. introduction", "\nintroduction", "\nkeywords", "\nindex terms"}
	restLower := strings.ToLower(rest)
	endPos := len(rest)
	for _, marker := range endMarkers {
		if pos := strings.Index(restLower, marker); pos != -1 && pos < endPos {
			endPos = pos
		}
	}
	abstract := strings.TrimSpace(rest[:endPos])
	abstract = strings.ReplaceAll(abstract, "\n", " ")
	if len(abstract) > 2500 {
		abstract = abstract[:2500]
	}
	return abstract
}

func normalizeSectionType(heading string) string {
	h := strings.ToLower(strings.TrimSpace(heading))
	switch {
	case strings.Contains(h, "abstract"):
		return "Abstract"
	case strings.Contains(h, "introduction") || strings.Contains(h, "background"):
		return "Introduction"
	case strings.Contains(h, "related") || strings.Contains(h, "preliminar"):
		return "Related Work"
	case strings.Contains(h, "method") || strings.Contains(h, "framework") || strings.Contains(h, "approach") || strings.Contains(h, "model") || strings.Contains(h, "architecture"):
		return "Methodology"
	case strings.Contains(h, "experiment") || strings.Contains(h, "evaluation") || strings.Contains(h, "ablation"):
		return "Experiments"
	case strings.Contains(h, "result") || strings.Contains(h, "discussion"):
		return "Results"
	case strings.Contains(h, "limitation"):
		return "Limitations"
	case strings.Contains(h, "conclusion") || strings.Contains(h, "future"):
		return "Conclusion"
	case strings.Contains(h, "reference") || strings.Contains(h, "bibliograph"):
		return "References"
	default:
		return "Section"
	}
}

func identifySections(pages []PageContent) []SectionContent {
	var sections []SectionContent
	currentTitle := "Overview & Abstract"
	currentType := "Abstract"
	startPage := 1
	var currentBuf strings.Builder
	order := 1

	for _, page := range pages {
		lines := strings.Split(page.CleanText, "\n")
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if len(trimmed) < 60 && sectionRegex.MatchString(trimmed) {
				if currentBuf.Len() > 0 {
					sections = append(sections, SectionContent{
						SectionTitle: currentTitle,
						SectionType:  currentType,
						StartPage:    startPage,
						EndPage:      page.PageNumber,
						Content:      strings.TrimSpace(currentBuf.String()),
						SectionOrder: order,
					})
					order++
					currentBuf.Reset()
				}
				currentTitle = trimmed
				currentType = normalizeSectionType(trimmed)
				startPage = page.PageNumber
				continue
			}
			currentBuf.WriteString(trimmed)
			currentBuf.WriteByte('\n')
		}
	}

	if currentBuf.Len() > 0 {
		endPage := 1
		if len(pages) > 0 {
			endPage = pages[len(pages)-1].PageNumber
		}
		sections = append(sections, SectionContent{
			SectionTitle: currentTitle,
			SectionType:  currentType,
			StartPage:    startPage,
			EndPage:      endPage,
			Content:      strings.TrimSpace(currentBuf.String()),
			SectionOrder: order,
		})
	}

	return sections
}

