package papercompare_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/GouTouZhao/PaperCompare/tools/pdfgen"
	"github.com/GouTouZhao/PaperCompare/tools/pdfparser"
	"github.com/GouTouZhao/PaperCompare/tools/snowflake"
	"github.com/GouTouZhao/PaperCompare/tools/token"
)

func TestSnowflakeAndToken(t *testing.T) {
	sf, err := snowflake.NewPodIPHashSnowflake()
	if err != nil {
		t.Fatalf("failed to create snowflake: %v", err)
	}
	id1, err := sf.NextID()
	if err != nil || id1 <= 0 {
		t.Fatalf("invalid id1: %d, err: %v", id1, err)
	}
	id2, err := sf.NextID()
	if err != nil || id2 <= id1 {
		t.Fatalf("expected id2 > id1, got %d vs %d", id2, id1)
	}

	access, refresh, err := token.GenerateTokens(t.Context(), nil, 10001, "demo@papercompare.org")
	if err != nil || access == "" || refresh == "" {
		t.Fatalf("GenerateTokens failed: %v", err)
	}
	claims, err := token.ParseAccessToken("Bearer " + access)
	if err != nil {
		t.Fatalf("ParseAccessToken failed: %v", err)
	}
	if claims.UserID != 10001 || claims.Email != "demo@papercompare.org" {
		t.Fatalf("unexpected claims: %+v", claims)
	}
}

func TestPDFGenAndParser(t *testing.T) {
	tmpPDF := filepath.Join(t.TempDir(), "sample_fedproto.pdf")
	err := pdfgen.GenerateSamplePDF(
		tmpPDF,
		"FedProto: Federated Prototype Learning across Heterogeneous Clients",
		"Yue Tan, Guodong Long",
		2022,
		"10.1609/aaai.v36i8.20819",
		[]pdfgen.SamplePage{
			{
				Heading: "Abstract",
				Lines: []string{
					"Abstract: We propose Federated Prototype Learning (FedProto) to communicate class prototypes instead of gradients.",
					"1 Introduction: Heterogeneous clients require prototype alignment.",
				},
			},
			{
				Heading: "2 Methodology",
				Lines: []string{
					"Each client computes class prototypes and aligns them with global prototypes via L2 regularization.",
				},
			},
		},
	)
	if err != nil {
		t.Fatalf("GenerateSamplePDF failed: %v", err)
	}

	parsed, err := pdfparser.ParsePDFFile(tmpPDF)
	if err != nil {
		t.Fatalf("ParsePDFFile failed: %v", err)
	}
	if parsed.PageCount != 2 {
		t.Fatalf("expected 2 pages, got %d", parsed.PageCount)
	}
	if !strings.Contains(parsed.DOI, "10.1609/aaai.v36i8.20819") {
		t.Fatalf("expected DOI extracted, got %q", parsed.DOI)
	}
	if len(parsed.Pages) != 2 || !strings.Contains(parsed.Pages[0].CleanText, "FedProto") {
		t.Fatalf("expected page 1 text to contain FedProto, got %q", parsed.Pages[0].CleanText)
	}
}

