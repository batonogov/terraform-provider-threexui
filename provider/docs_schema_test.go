package provider

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

// TestDocumentedSchemaSections compares opted-in, hand-written documentation
// sections with the live provider schema. It deliberately checks identifiers,
// nesting, and block shape instead of prose or field order, so translations and
// editorial improvements do not make the guard brittle. Add another assertion
// when a hand-written nested schema section needs the same protection.
func TestDocumentedSchemaSections(t *testing.T) {
	observatory := xrayObservatorySchema()
	assertDocumentedBlock(t, "docs/resources/xray_observatory.md", "### observatory", observatory.Blocks["observatory"])
	assertDocumentedBlock(t, "docs/resources/xray_observatory.md", "### burst_observatory", observatory.Blocks["burst_observatory"])

	burst := requireListNestedBlock(t, observatory.Blocks["burst_observatory"])
	assertDocumentedBlock(t, "docs/resources/xray_observatory.md", "#### ping_config", burst.NestedObject.Blocks["ping_config"])

	inbound := inboundSettingsBlockSchemas()
	assertDocumentedBlock(t, "docs/resources/inbound.md", "#### `mtproto_settings`", inbound["mtproto_settings"])

	mtproto := requireSingleNestedBlock(t, inbound["mtproto_settings"])
	assertDocumentedBlock(t, "docs/resources/inbound.md", "##### `domain_fronting`", mtproto.Blocks["domain_fronting"])

	assertDocumentedBlock(t, "docs/resources/inbound.md", "#### `amneziawg_settings`", inbound["amneziawg_settings"])
	amneziawg := requireSingleNestedBlock(t, inbound["amneziawg_settings"])
	assertDocumentedBlock(t, "docs/resources/inbound.md", "##### `server`", amneziawg.Blocks["server"])
	assertDocumentedBlock(t, "docs/resources/inbound.md", "##### `clients`", amneziawg.Blocks["clients"])

	assertDocumentedBlock(t, "docs/resources/inbound.md", "#### `tuic_settings`", inbound["tuic_settings"])
	// The nested `server`/`clients` headings deliberately reuse the
	// AmneziaWG spelling and documentedFields matches the FIRST heading, so
	// only the parent block is guarded here; the field lists above carry the
	// per-attribute documentation.
}

func TestInboundProtocolDescriptionListsVersionedProtocols(t *testing.T) {
	var resp resource.SchemaResponse
	(&InboundResource{}).Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("building inbound schema: %v", resp.Diagnostics)
	}

	protocol, ok := resp.Schema.Attributes["protocol"].(schema.StringAttribute)
	if !ok {
		t.Fatalf("protocol has unexpected schema type %T", resp.Schema.Attributes["protocol"])
	}
	for _, want := range []string{"tun", "v3.2.7", "mtproto", "v3.3.0"} {
		if !strings.Contains(protocol.Description, want) {
			t.Errorf("protocol description must contain %q: %s", want, protocol.Description)
		}
	}
}

func assertDocumentedBlock(t *testing.T, docPath, heading string, block schema.Block) {
	t.Helper()

	var (
		attributes map[string]schema.Attribute
		blocks     map[string]schema.Block
		kind       string
	)
	switch typed := block.(type) {
	case schema.ListNestedBlock:
		attributes = typed.NestedObject.Attributes
		blocks = typed.NestedObject.Blocks
		kind = "Block List"
	case schema.SingleNestedBlock:
		attributes = typed.Attributes
		blocks = typed.Blocks
		kind = "Block"
	default:
		t.Fatalf("%s has unsupported schema block type %T", heading, block)
	}

	want := make([]string, 0, len(attributes)+len(blocks))
	for name := range attributes {
		want = append(want, name)
	}
	for name := range blocks {
		want = append(want, name)
	}
	sort.Strings(want)

	wantHeading := heading + " (Optional, " + kind
	got := documentedFields(t, docPath, wantHeading)
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("%s fields do not match schema\n got: %v\nwant: %v", wantHeading, got, want)
	}
}

func documentedFields(t *testing.T, docPath, heading string) []string {
	t.Helper()

	data, err := os.ReadFile(filepath.Join("..", docPath))
	if err != nil {
		t.Fatalf("read %s: %v", docPath, err)
	}

	lines := strings.Split(string(data), "\n")
	start := -1
	for i, line := range lines {
		remainder := strings.TrimPrefix(line, heading)
		if remainder != line && (strings.HasPrefix(remainder, ")") || strings.HasPrefix(remainder, ",")) {
			start = i + 1
			break
		}
	}
	if start == -1 {
		t.Fatalf("%s: missing heading %q", docPath, heading)
	}

	fields := make([]string, 0)
	for _, line := range lines[start:] {
		if headingLevel(line) > 0 {
			break
		}
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "- `") {
			continue
		}
		nameAndRest := strings.TrimPrefix(trimmed, "- `")
		end := strings.IndexByte(nameAndRest, '`')
		if end == -1 {
			continue
		}
		fields = append(fields, nameAndRest[:end])
	}
	sort.Strings(fields)
	return fields
}

func headingLevel(line string) int {
	trimmed := strings.TrimLeft(line, "#")
	level := len(line) - len(trimmed)
	if level == 0 || len(trimmed) == 0 || trimmed[0] != ' ' {
		return 0
	}
	return level
}

func requireListNestedBlock(t *testing.T, block schema.Block) schema.ListNestedBlock {
	t.Helper()
	typed, ok := block.(schema.ListNestedBlock)
	if !ok {
		t.Fatalf("expected ListNestedBlock, got %T", block)
	}
	return typed
}

func requireSingleNestedBlock(t *testing.T, block schema.Block) schema.SingleNestedBlock {
	t.Helper()
	typed, ok := block.(schema.SingleNestedBlock)
	if !ok {
		t.Fatalf("expected SingleNestedBlock, got %T", block)
	}
	return typed
}
