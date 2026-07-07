//
// Copyright (C) 2026 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/arcnet-cli
//

package core

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/yuin/goldmark"
	meta "github.com/yuin/goldmark-meta"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	gmtext "github.com/yuin/goldmark/text"
	"gopkg.in/yaml.v3"
)

// errNoCause is passed to faults.Type.With for guard conditions that are not
// caused by an underlying Go error, so the rendered message has no trailing
// "%!s(<nil>)" artifact.
var errNoCause = errors.New("")

func newMarkdownParser() goldmark.Markdown {
	return goldmark.New(goldmark.WithExtensions(meta.Meta))
}

// parseDocument parses source into a goldmark AST plus its YAML front-matter
// manifest map, keeping goldmark's own AST types confined to this file
// (research.md D2, D3).
func parseDocument(source []byte) (ast.Node, map[string]any, error) {
	md := newMarkdownParser()
	ctx := parser.NewContext()
	doc := md.Parser().Parse(gmtext.NewReader(source), parser.WithContext(ctx))

	m, err := meta.TryGet(ctx)
	if err != nil {
		return nil, nil, err
	}

	return doc, normalizeYAMLMap(m), nil
}

// ParsePatch parses a CORE §12.2 document patch: the front-matter manifest,
// plus every H1-type/H2-node body section.
func ParsePatch(r io.Reader) (Patch, error) {
	source, err := io.ReadAll(r)
	if err != nil {
		return Patch{}, err
	}

	doc, manifest, err := parseDocument(source)
	if err != nil {
		return Patch{}, ErrManifestInvalid.With(err)
	}

	patch, err := decodePatchManifest(manifest)
	if err != nil {
		return Patch{}, err
	}

	nodes, err := parsePatchBody(doc, source)
	if err != nil {
		return Patch{}, err
	}
	patch.Nodes = nodes

	return patch, nil
}

// ParseNode parses one on-disk graph node file (front-matter + body) into a
// Node.
func ParseNode(r io.Reader) (Node, error) {
	source, err := io.ReadAll(r)
	if err != nil {
		return Node{}, err
	}

	doc, manifest, err := parseDocument(source)
	if err != nil {
		return Node{}, ErrManifestInvalid.With(err)
	}

	id, typ, manifest, err := identityFields(manifest)
	if err != nil {
		return Node{}, ErrManifestInvalid.With(err)
	}
	// AST §4: a standalone node file's "@id" MUST equal its own filename
	// (basename, extension stripped) — but ParseNode's signature is pinned
	// with no filename parameter (contracts/ast-contract.md), so it cannot
	// perform that comparison itself. That check is enforced by the caller
	// (internal/app/graph/service.Apply), which does have the filename;
	// everything ParseNode can validate from the document's own bytes alone
	// (legacy "kind" field, absent/empty "@id"/"@type") is validated here.

	published, manifest := extractPublished(manifest)
	attrs := wrapAttrs(manifest)

	children := childSlice(doc)
	if len(children) > 0 {
		if h, ok := children[0].(*ast.Heading); ok && h.Level == 1 {
			children = children[1:]
		}
	}

	texts, hrefs, edges := walkNodeBody(children, source, typ)

	return Node{
		ID:        id,
		Type:      typ,
		Published: published,
		Attrs:     attrs,
		Texts:     texts,
		HRefs:     hrefs,
		Edges:     edges,
	}, nil
}

// identityFields extracts and validates a node's mandatory "@id"/"@type"
// fields out of its own front-matter/yaml-fence map (research.md D7,
// contracts/ast-contract.md "Old-format rejection"): a legacy "kind" key
// present at all, or an absent/empty "@id"/"@type", is rejected with a
// descriptive error before any body-walking begins — no partial Node is
// ever constructed. Returns the remaining manifest with "@id"/"@type"
// removed, ready for extractPublished/wrapAttrs.
func identityFields(manifest map[string]any) (id, typ string, rest map[string]any, err error) {
	if _, hasKind := manifest["kind"]; hasKind {
		return "", "", nil, fmt.Errorf("legacy %q field present, expected \"@id\"/\"@type\"", "kind")
	}

	id, _ = manifest["@id"].(string)
	if id == "" {
		return "", "", nil, fmt.Errorf("missing mandatory %q field", "@id")
	}

	typ, _ = manifest["@type"].(string)
	if typ == "" {
		return "", "", nil, fmt.Errorf("missing mandatory %q field", "@type")
	}

	rest = make(map[string]any, len(manifest))
	for k, v := range manifest {
		if k == "@id" || k == "@type" {
			continue
		}
		rest[k] = v
	}
	return id, typ, rest, nil
}

// patchNodeIdentity resolves one patch node section's "@id"/"@type"
// (BUG-001): unlike a standalone file, a patch section's "## <ID>" heading
// and enclosing "# <Type>" heading satisfy "@id"/"@type" by themselves —
// CORE §12.2's own convention, and the shape every pre-existing patch
// fixture (and real external patch producers, e.g. fogfish/bots) already
// use. An explicit "@id"/"@type" key inside the node's own yaml fence is
// optional; if present, it MUST agree with the corresponding heading or
// the contribution is rejected as inconsistent. A legacy "kind" key
// present at all is still rejected unconditionally, exactly like
// identityFields. Returns the remaining manifest with "@id"/"@type"
// removed (if present), ready for extractPublished/wrapAttrs.
func patchNodeIdentity(manifest map[string]any, idHeading, typeHeading string) (id, typ string, rest map[string]any, err error) {
	if _, hasKind := manifest["kind"]; hasKind {
		return "", "", nil, fmt.Errorf("legacy %q field present, expected \"@id\"/\"@type\"", "kind")
	}

	id = idHeading
	if explicit, ok := manifest["@id"].(string); ok && explicit != "" {
		if explicit != idHeading {
			return "", "", nil, fmt.Errorf("\"@id\" %q does not match section heading %q", explicit, idHeading)
		}
		id = explicit
	}
	if id == "" {
		return "", "", nil, fmt.Errorf("missing mandatory %q field", "@id")
	}

	typ = strings.ToLower(typeHeading)
	if explicit, ok := manifest["@type"].(string); ok && explicit != "" {
		if !strings.EqualFold(explicit, typeHeading) {
			return "", "", nil, fmt.Errorf("\"@type\" %q does not match section heading %q", explicit, typeHeading)
		}
		typ = explicit
	}
	if typ == "" {
		return "", "", nil, fmt.Errorf("missing mandatory %q field", "@type")
	}

	rest = make(map[string]any, len(manifest))
	for k, v := range manifest {
		if k == "@id" || k == "@type" {
			continue
		}
		rest[k] = v
	}
	return id, typ, rest, nil
}

// extractPublished pulls a "published" key out of a raw front-matter/yaml-
// fence map, decoding it via the same decodeManifestDate decodePatchManifest
// already uses, and returns the remaining map with that key removed
// (research.md D2) — used by both ParseNode and parsePatchBody's per-node
// construction, so "published" is never left behind as a generic Attrs key.
func extractPublished(manifest map[string]any) (time.Time, map[string]any) {
	published, _ := decodeManifestDate(manifest["published"])
	if _, ok := manifest["published"]; !ok {
		return published, manifest
	}
	out := make(map[string]any, len(manifest)-1)
	for k, v := range manifest {
		if k == "published" {
			continue
		}
		out[k] = v
	}
	return published, out
}

// wrapAttrs converts a raw front-matter/yaml-fence map (with "@id"/"@type"/
// "published" already removed) into Attrs' map[string][]Predicate shape
// (research.md D3): a YAML scalar value becomes a one-element list; a YAML
// sequence becomes one Predicate per element, in order.
func wrapAttrs(manifest map[string]any) map[string][]Predicate {
	if len(manifest) == 0 {
		return nil
	}
	attrs := make(map[string][]Predicate, len(manifest))
	for k, v := range manifest {
		attrs[k] = wrapPredicateValue(v)
	}
	return attrs
}

func wrapPredicateValue(v any) []Predicate {
	if seq, ok := v.([]any); ok {
		out := make([]Predicate, 0, len(seq))
		for _, elem := range seq {
			out = append(out, Predicate{Value: elem})
		}
		return out
	}
	return []Predicate{{Value: v}}
}

// textPredicateFor is a small, explicitly temporary "@type"->text-predicate
// lookup table (research.md D4): it names the leading and trailing prose
// slots walkNodeBody still recognizes structurally, so a node's stored
// prose keys are domain-appropriate (a source's leading prose really is its
// "abstract") rather than the old fixed "text"/"notes" pair. This is a
// stopgap superseded by spec 011's Schema Index, which will derive text
// predicate names from a graph's actual schema instead of a hardcoded
// table.
func textPredicateFor(nodeType string, leading bool) string {
	if !leading {
		return "notes"
	}
	switch nodeType {
	case "source":
		return "abstract"
	case "entity":
		return "definition"
	case "resource":
		return "relevance"
	case "hypothesis":
		return "claim"
	case "aporia":
		return "tension"
	case "thought":
		return "claim"
	default:
		return "text"
	}
}

func decodePatchManifest(manifest map[string]any) (Patch, error) {
	if kindValue, _ := manifest["kind"].(string); kindValue != "patch" {
		return Patch{}, ErrManifestInvalid.With(errNoCause)
	}

	document, _ := manifest["document"].(string)
	if document == "" {
		return Patch{}, ErrManifestInvalid.With(errNoCause)
	}

	published, ok := decodeManifestDate(manifest["published"])
	if !ok {
		return Patch{}, ErrManifestInvalid.With(errNoCause)
	}

	title, _ := manifest["title"].(string)

	var stats map[string]any
	if s, ok := manifest["stats"].(map[string]any); ok {
		stats = s
	}

	return Patch{
		Document:  document,
		Published: published,
		Title:     title,
		Stats:     stats,
	}, nil
}

func decodeManifestDate(v any) (time.Time, bool) {
	switch val := v.(type) {
	case time.Time:
		return val, true
	case string:
		for _, layout := range []string{"2006-01-02", time.RFC3339, "2006-01-02T15:04:05Z"} {
			if t, err := time.Parse(layout, val); err == nil {
				return t, true
			}
		}
	}
	return time.Time{}, false
}

// normalizeYAMLMap converts goldmark-meta's yaml.v2-flavored nested maps
// (map[interface{}]interface{}) into map[string]any/[]any consistently, so
// downstream Attrs/Stats trees have uniform, JSON/YAML-friendly types
// regardless of nesting depth.
func normalizeYAMLMap(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = normalizeYAMLValue(v)
	}
	return out
}

func normalizeYAMLValue(v any) any {
	switch val := v.(type) {
	case map[any]any:
		out := make(map[string]any, len(val))
		for k, vv := range val {
			out[fmt.Sprint(k)] = normalizeYAMLValue(vv)
		}
		return out
	case map[string]any:
		return normalizeYAMLMap(val)
	case []any:
		out := make([]any, len(val))
		for i, vv := range val {
			out[i] = normalizeYAMLValue(vv)
		}
		return out
	default:
		return v
	}
}

// childSlice materializes a node's block-level children as a slice, for
// lookahead-based section-boundary detection.
func childSlice(n ast.Node) []ast.Node {
	var out []ast.Node
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		out = append(out, c)
	}
	return out
}

func linesText(n ast.Node, source []byte) string {
	lines := n.Lines()
	if lines == nil || lines.Len() == 0 {
		return ""
	}
	parts := make([]string, 0, lines.Len())
	for i := 0; i < lines.Len(); i++ {
		seg := lines.At(i)
		parts = append(parts, strings.TrimSpace(string(seg.Value(source))))
	}
	return strings.TrimSpace(strings.Join(parts, " "))
}

// parsePatchBody walks a patch's body: one or more H1-type sections, each
// containing one or more H2-node sections (a heading immediately followed
// by a fenced ```yaml block), per CORE §12.2/research.md D3. Per BUG-001,
// each node's "@id"/"@type" are satisfied by its own "## <ID>" heading and
// the enclosing "# <Type>" heading — CORE §12.2's own convention, and the
// shape every pre-existing patch fixture (and real external patch
// producers) already use — an explicit "@id"/"@type" key inside the node's
// own yaml fence is optional, and if present MUST agree with the
// corresponding heading (see patchNodeIdentity).
func parsePatchBody(doc ast.Node, source []byte) ([]Node, error) {
	children := childSlice(doc)
	if len(children) == 0 {
		return nil, ErrPatchStructure.With(errNoCause)
	}

	var nodes []Node
	i := 0
	for i < len(children) {
		h1, ok := children[i].(*ast.Heading)
		if !ok || h1.Level != 1 {
			return nil, ErrPatchStructure.With(errNoCause)
		}
		typeHeading := linesText(h1, source)
		i++

		sawNode := false
		for i < len(children) {
			h2, ok := children[i].(*ast.Heading)
			if !ok || h2.Level != 2 {
				break
			}
			if i+1 >= len(children) {
				return nil, ErrPatchStructure.With(errNoCause)
			}
			fence, ok := children[i+1].(*ast.FencedCodeBlock)
			if !ok || !isYAMLFence(fence, source) {
				return nil, ErrPatchStructure.With(errNoCause)
			}

			idHeading := linesText(h2, source)
			manifest, err := decodeYAMLBlock(fence, source)
			if err != nil {
				return nil, ErrPatchStructure.With(err)
			}

			id, typ, manifest, err := patchNodeIdentity(manifest, idHeading, typeHeading)
			if err != nil {
				return nil, ErrManifestInvalid.With(err)
			}

			published, manifest := extractPublished(manifest)
			attrs := wrapAttrs(manifest)

			i += 2

			start := i
			for i < len(children) && !isSectionBoundary(children, i, source) {
				i++
			}

			texts, hrefs, edges := walkNodeBody(children[start:i], source, typ)

			nodes = append(nodes, Node{
				ID:        id,
				Type:      typ,
				Published: published,
				Attrs:     attrs,
				Texts:     texts,
				HRefs:     hrefs,
				Edges:     edges,
			})
			sawNode = true
		}
		if !sawNode {
			return nil, ErrPatchStructure.With(errNoCause)
		}
	}

	return nodes, nil
}

func isYAMLFence(fence *ast.FencedCodeBlock, source []byte) bool {
	return strings.TrimSpace(string(fence.Language(source))) == "yaml"
}

// isSectionBoundary reports whether children[i] opens a new H1-type section
// or a new H2-node section (heading immediately followed by a yaml fence) —
// as opposed to an H2 heading with no yaml fence following, which is a
// link-block heading nested within the current node's body.
func isSectionBoundary(children []ast.Node, i int, source []byte) bool {
	h, ok := children[i].(*ast.Heading)
	if !ok {
		return false
	}
	if h.Level == 1 {
		return true
	}
	if h.Level == 2 && i+1 < len(children) {
		if fence, ok := children[i+1].(*ast.FencedCodeBlock); ok && isYAMLFence(fence, source) {
			return true
		}
	}
	return false
}

func decodeYAMLBlock(fence *ast.FencedCodeBlock, source []byte) (map[string]any, error) {
	raw := fence.Lines().Value(source)
	if len(bytes.TrimSpace(raw)) == 0 {
		return map[string]any{}, nil
	}

	var attrs map[string]any
	if err := yaml.Unmarshal(raw, &attrs); err != nil {
		return nil, err
	}
	if attrs == nil {
		attrs = map[string]any{}
	}
	return attrs, nil
}

// walkNodeBody parses a node's body span (everything after its identity
// heading/yaml-fence, for a patch; everything after the derived H1 title,
// for an on-disk node file) into Texts/HRefs/Edges per AST §6: leading
// prose, an optional bare edge list, zero or more heading+list link blocks,
// then trailing prose. The structural recognition (leading paragraphs /
// optional bare list / heading-or-bold-label-plus-list blocks / trailing
// paragraphs) is unchanged from specs/003; only what it produces changed
// (research.md D4/D5): the two prose slots are now named via
// textPredicateFor instead of two fixed fields, and every link — whether
// from the bare list or from a heading/bold-label block — flattens into one
// Edges slice, in the order encountered, with no grouping key retained.
func walkNodeBody(children []ast.Node, source []byte, nodeType string) (texts map[string]string, hrefs, edges []Link) {
	idx := 0

	var leading []string
	for idx < len(children) {
		p, ok := children[idx].(*ast.Paragraph)
		if !ok {
			break
		}
		// A bold-label paragraph (BUG-003) immediately followed by a list
		// opens a predicate-grouped block, not more leading prose — leave
		// it for the headed/labeled-blocks loop below to claim, so its
		// list is captured as Edges entries rather than swept into Texts
		// or misclassified as the ungrouped bare-edges list.
		if _, isLabel := boldLabel(p, source); isLabel && idx+1 < len(children) {
			if _, isList := children[idx+1].(*ast.List); isList {
				break
			}
		}
		leading = append(leading, linesText(p, source))
		idx++
	}
	rawText := strings.Join(leading, "\n\n")

	if idx < len(children) {
		if list, ok := children[idx].(*ast.List); ok {
			edges = collectListLinks(list, source)
			idx++
		}
	}

	for idx < len(children) {
		_, matched := blockTitle(children[idx], source)
		if !matched || idx+1 >= len(children) {
			break
		}
		list, ok := children[idx+1].(*ast.List)
		if !ok {
			break
		}

		edges = append(edges, collectListLinks(list, source)...)
		idx += 2
	}

	// A List reaching this point (BUG-003) means the patch's body did not
	// pair a heading/bold-label title with it the way the loop above
	// expects (e.g. a bare list with no title at all, following an
	// already-matched block) — fold it into the ungrouped edges rather
	// than silently discarding it, so no declared relation is ever lost.
	var trailing []string
	for idx < len(children) {
		switch v := children[idx].(type) {
		case *ast.Paragraph:
			trailing = append(trailing, linesText(v, source))
		case *ast.List:
			edges = append(edges, collectListLinks(v, source)...)
		}
		idx++
	}
	rawNotes := strings.Join(trailing, "\n\n")

	strippedText, textHRefs := extractInlineLinks(rawText)
	strippedNotes, notesHRefs := extractInlineLinks(rawNotes)

	hrefs = append(hrefs, textHRefs...)
	hrefs = append(hrefs, notesHRefs...)

	texts = map[string]string{}
	if strippedText != "" {
		texts[textPredicateFor(nodeType, true)] = strippedText
	}
	if strippedNotes != "" {
		texts[textPredicateFor(nodeType, false)] = strippedNotes
	}
	if len(texts) == 0 {
		texts = nil
	}

	return texts, hrefs, edges
}

func collectListLinks(list *ast.List, source []byte) []Link {
	var out []Link
	for c := list.FirstChild(); c != nil; c = c.NextSibling() {
		item, ok := c.(*ast.ListItem)
		if !ok {
			continue
		}
		line := listItemText(item, source)
		if l, ok := parseListItemLink(line); ok {
			out = append(out, l)
		}
	}
	return out
}

func listItemText(item ast.Node, source []byte) string {
	var parts []string
	for c := item.FirstChild(); c != nil; c = c.NextSibling() {
		if t := linesText(c, source); t != "" {
			parts = append(parts, t)
		}
	}
	return strings.TrimSpace(strings.Join(parts, " "))
}

// blockTitle reports the display title of a predicate-grouped block's
// opening node, in either of the two forms research.md D3c permits
// (BUG-003): a "## Label" H2 heading (this feature's own convention), or
// a "**Label**" bold-text paragraph — CORE §12.2's canonical convention
// ("node bodies use bold labels, never headings"). Both are recognized so
// a patch node's body may freely use either or mix both across blocks. The
// title text itself is no longer retained anywhere (research.md D5 —
// grouping is derived at render time, never stored), only used to decide
// whether the following list belongs to a link block at all.
func blockTitle(n ast.Node, source []byte) (string, bool) {
	switch v := n.(type) {
	case *ast.Heading:
		return linesText(v, source), true
	case *ast.Paragraph:
		return boldLabel(v, source)
	default:
		return "", false
	}
}

var boldLabelPattern = regexp.MustCompile(`^\*\*([^*]+)\*\*$`)

// boldLabel reports whether p's entire raw text is a single bold-emphasis
// run ("**Label**") with nothing else alongside it.
func boldLabel(p *ast.Paragraph, source []byte) (string, bool) {
	m := boldLabelPattern.FindStringSubmatch(linesText(p, source))
	if m == nil {
		return "", false
	}
	return strings.TrimSpace(m[1]), true
}

var listItemPattern = regexp.MustCompile(`^(?:(\w+)::\s*)?\[\[([^\]|]+?)(?:\|([^\]]+?))?\]\]$`)

func parseListItemLink(line string) (Link, bool) {
	m := listItemPattern.FindStringSubmatch(line)
	if m == nil {
		return Link{}, false
	}
	return Link{Predicate: m[1], Target: m[2], Alias: m[3]}, true
}

// inlineLinkPattern recognizes, within already-isolated prose text, either
// the inline predicate form "[predicate:: [[Target]]]" or a bare/aliased
// wikilink "[[Target]]"/"[[Target|alias]]" (research.md D3/D3b).
var inlineLinkPattern = regexp.MustCompile(`\[(\w+)::\s*\[\[([^\]|]+?)(?:\|([^\]]+?))?\]\]\]|\[\[([^\]|]+?)(?:\|([^\]]+?))?\]\]`)

func extractInlineLinks(text string) (string, []Link) {
	matches := inlineLinkPattern.FindAllStringSubmatchIndex(text, -1)
	if len(matches) == 0 {
		return text, nil
	}

	var hrefs []Link
	var b strings.Builder
	last := 0
	for _, m := range matches {
		b.WriteString(text[last:m[0]])

		var link Link
		if m[2] >= 0 {
			link.Predicate = text[m[2]:m[3]]
			link.Target = text[m[4]:m[5]]
			if m[6] >= 0 {
				link.Alias = text[m[6]:m[7]]
			}
		} else {
			link.Target = text[m[8]:m[9]]
			if m[10] >= 0 {
				link.Alias = text[m[10]:m[11]]
			}
		}

		hrefs = append(hrefs, link)
		if link.Alias != "" {
			b.WriteString(link.Alias)
		} else {
			b.WriteString(link.Target)
		}
		last = m[1]
	}
	b.WriteString(text[last:])

	return b.String(), hrefs
}

// RenderNode serializes n back to Markdown: front-matter ("@id"/"@type"
// first, then sorted attribute keys) + Texts + Edges (one flat bulleted
// list), per contracts/ast-contract.md. Inline wikilink markup is
// reconstructed into Texts values from HRefs (research.md D3b).
func RenderNode(n Node) ([]byte, error) {
	frontMatter, err := renderFrontMatter(n)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	buf.WriteString("---\n")
	buf.Write(frontMatter)
	buf.WriteString("---\n")
	buf.WriteString("# " + n.ID + "\n")
	buf.Write(renderNodeBody(n))

	return buf.Bytes(), nil
}

// renderNodeBody renders n's Texts/Edges (with HRefs reconstructed into
// Texts), shared verbatim by RenderNode's on-disk single-node shape and
// RenderPatch's per-node patch-exchange section (specs/007-arc-subgraph,
// research.md D2/D9) — the only difference between the two callers is what
// precedes this body (a "# <ID>" H1 heading vs. a "## <ID>" H2 heading plus
// a fenced yaml block).
//
// Physical layout (contracts/ast-contract.md: "matching the original
// leading-prose/edges/trailing-prose visual layout"): the leading-slot key
// (textPredicateFor(n.Type, true)) renders first if present, then any other
// Texts keys sorted alphabetically, then Edges as one flat bulleted list
// (research.md D6 — no "## <Label>" grouped rendering), then the
// trailing-slot key (textPredicateFor(n.Type, false)) last if present. This
// ordering is load-bearing, not cosmetic: walkNodeBody's structural parser
// only recognizes leading-paragraphs/list/trailing-paragraphs in that
// physical sequence, so rendering every Texts value before Edges (rather
// than sandwiching Edges between the leading and trailing slots) would
// break FR-014's round-trip by merging the trailing prose back into the
// leading slot on re-parse.
func renderNodeBody(n Node) []byte {
	var buf bytes.Buffer

	consumed := make([]bool, len(n.HRefs))
	writeText := func(key string) {
		rendered := reconstructHRefs(n.Texts[key], n.HRefs, consumed)
		if rendered == "" {
			return
		}
		buf.WriteString("\n")
		buf.WriteString(rendered)
		buf.WriteString("\n")
	}

	leadingKey := textPredicateFor(n.Type, true)
	trailingKey := textPredicateFor(n.Type, false)

	if _, ok := n.Texts[leadingKey]; ok {
		writeText(leadingKey)
	}

	var other []string
	for k := range n.Texts {
		if k == leadingKey || k == trailingKey {
			continue
		}
		other = append(other, k)
	}
	sort.Strings(other)
	for _, k := range other {
		writeText(k)
	}

	if len(n.Edges) > 0 {
		buf.WriteString("\n")
		for _, e := range n.Edges {
			buf.WriteString(renderLinkBullet(e))
			buf.WriteString("\n")
		}
	}

	if _, ok := n.Texts[trailingKey]; ok {
		writeText(trailingKey)
	}

	return buf.Bytes()
}

// RenderPatch is the structural inverse of ParsePatch (research.md D2): a
// `---`-delimited manifest (kind: patch, document, published, title,
// stats), then p.Nodes grouped by Type (sorted alphabetically) under
// "# <Type>" headings, each node (sorted alphabetically by ID within its
// type — research.md D9) under a "## <ID>" heading with a fenced yaml block
// (attributes plus "@id"/"@type" — parsePatchBody reads "@type" from each
// node's own fence, not from the enclosing H1, so it must be present there
// too, unlike the old "kind"-omitted-under-H1 convention) and its body via
// the same renderNodeBody RenderNode uses.
func RenderPatch(p Patch) ([]byte, error) {
	manifest, err := renderPatchManifest(p)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	buf.WriteString("---\n")
	buf.Write(manifest)
	buf.WriteString("---\n")

	for i, typ := range sortedPatchTypes(p.Nodes) {
		nodes := nodesOfType(p.Nodes, typ)

		if i > 0 {
			buf.WriteString("\n")
		}
		buf.WriteString("# " + titleCaseType(typ) + "\n")

		for _, n := range nodes {
			fence, err := renderAttrYAML(n.ID, n.Type, n.Published, n.Attrs)
			if err != nil {
				return nil, err
			}

			buf.WriteString("\n## " + n.ID + "\n")
			buf.WriteString("```yaml\n")
			buf.Write(fence)
			buf.WriteString("```\n")
			buf.Write(renderNodeBody(n))
		}
	}

	return buf.Bytes(), nil
}

// sortedPatchTypes returns every distinct Type present in nodes, sorted
// alphabetically (research.md D9).
func sortedPatchTypes(nodes []Node) []string {
	seen := map[string]bool{}
	var types []string
	for _, n := range nodes {
		if !seen[n.Type] {
			seen[n.Type] = true
			types = append(types, n.Type)
		}
	}
	sort.Strings(types)
	return types
}

// nodesOfType returns every node of typ, sorted alphabetically by ID
// (research.md D9).
func nodesOfType(nodes []Node, typ string) []Node {
	var out []Node
	for _, n := range nodes {
		if n.Type == typ {
			out = append(out, n)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// titleCaseType renders a Type for display as a section heading ("entity"
// -> "Entity") — this is purely a cosmetic organizational label now
// (parsePatchBody reads each node's actual Type from its own yaml fence,
// not from this heading), so this casing choice is not load-bearing for the
// round-trip property.
func titleCaseType(t string) string {
	if t == "" {
		return t
	}
	r := []rune(t)
	return strings.ToUpper(string(r[0])) + string(r[1:])
}

// renderPatchManifest renders p's document-level manifest as a mapping:
// kind: patch, document, published (date-only, "2006-01-02"), title (when
// non-empty), and stats (when non-empty, flow-style — "{a: 1, b: 2}") to
// match cli-contract.md's example shape.
func renderPatchManifest(p Patch) ([]byte, error) {
	root := &yaml.Node{Kind: yaml.MappingNode}

	if err := appendYAMLPair(root, "kind", "patch"); err != nil {
		return nil, err
	}
	if err := appendYAMLPair(root, "document", p.Document); err != nil {
		return nil, err
	}
	if err := appendYAMLPair(root, "published", p.Published.Format("2006-01-02")); err != nil {
		return nil, err
	}
	if p.Title != "" {
		if err := appendYAMLPair(root, "title", p.Title); err != nil {
			return nil, err
		}
	}
	if len(p.Stats) > 0 {
		keyNode, err := encodeYAMLNode("stats")
		if err != nil {
			return nil, err
		}
		statsNode, err := encodeYAMLNode(p.Stats)
		if err != nil {
			return nil, err
		}
		statsNode.Style = yaml.FlowStyle
		root.Content = append(root.Content, keyNode, statsNode)
	}

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	if err := enc.Encode(root); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func renderLinkBullet(l Link) string {
	return "- " + markupFor(l)
}

func markupFor(l Link) string {
	target := l.Target
	inner := "[[" + target + "]]"
	if l.Alias != "" {
		inner = "[[" + target + "|" + l.Alias + "]]"
	}
	if l.Predicate != "" {
		return l.Predicate + ":: " + inner
	}
	return inner
}

func renderFrontMatter(n Node) ([]byte, error) {
	return renderAttrYAML(n.ID, n.Type, n.Published, n.Attrs)
}

// renderAttrYAML renders a YAML mapping: "@id" and "@type" first (both
// quoted YAML keys — a leading "@" is a reserved plain-scalar indicator, so
// these keys must be rendered with explicit quoting to stay valid,
// unambiguous YAML), then every other Attrs key sorted alphabetically (a
// single-element []Predicate as a bare scalar, a multi-element list as a
// YAML sequence — research.md D3), then "published" last when non-zero.
// Shared by RenderNode's front matter and RenderPatch's per-node fence
// (research.md D2/D9), so both stay the single, structurally correct place
// this shape is produced.
func renderAttrYAML(id, typ string, published time.Time, attrs map[string][]Predicate) ([]byte, error) {
	root := &yaml.Node{Kind: yaml.MappingNode}

	if err := appendQuotedKeyYAMLPair(root, "@id", id); err != nil {
		return nil, err
	}
	if err := appendQuotedKeyYAMLPair(root, "@type", typ); err != nil {
		return nil, err
	}

	keys := make([]string, 0, len(attrs))
	for k := range attrs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		if err := appendYAMLPair(root, k, encodePredicateList(attrs[k])); err != nil {
			return nil, err
		}
	}

	if !published.IsZero() {
		if err := appendYAMLPair(root, "published", published.Format("2006-01-02")); err != nil {
			return nil, err
		}
	}

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	if err := enc.Encode(root); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// encodePredicateList collapses a single-element []Predicate back to its
// bare scalar Value, and renders a multi-element list as a plain []any
// sequence of each element's Value (research.md D3) — the render-time
// inverse of wrapPredicateValue.
func encodePredicateList(preds []Predicate) any {
	if len(preds) == 1 {
		return preds[0].Value
	}
	values := make([]any, len(preds))
	for i, p := range preds {
		values[i] = p.Value
	}
	return values
}

func appendYAMLPair(root *yaml.Node, key string, value any) error {
	keyNode, err := encodeYAMLNode(key)
	if err != nil {
		return err
	}
	valNode, err := encodeYAMLNode(value)
	if err != nil {
		return err
	}
	root.Content = append(root.Content, keyNode, valNode)
	return nil
}

// appendQuotedKeyYAMLPair is appendYAMLPair's variant for a key that must be
// rendered with explicit double-quote styling ("@id"/"@type" — a leading
// "@" is a YAML reserved indicator character in a plain scalar).
func appendQuotedKeyYAMLPair(root *yaml.Node, key string, value any) error {
	keyNode := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key, Style: yaml.DoubleQuotedStyle}
	valNode, err := encodeYAMLNode(value)
	if err != nil {
		return err
	}
	root.Content = append(root.Content, keyNode, valNode)
	return nil
}

func encodeYAMLNode(v any) (*yaml.Node, error) {
	var n yaml.Node
	if err := n.Encode(v); err != nil {
		return nil, err
	}
	return &n, nil
}

type span struct{ start, end int }

var existingBracketPattern = regexp.MustCompile(`\[+[^\[\]]*\]+`)

func existingBracketSpans(text string) []span {
	matches := existingBracketPattern.FindAllStringIndex(text, -1)
	spans := make([]span, 0, len(matches))
	for _, m := range matches {
		spans = append(spans, span{start: m[0], end: m[1]})
	}
	return spans
}

// reconstructHRefs re-inserts bracket markup for each not-yet-consumed href
// whose display substring occurs eligibly in text (research.md D3b,
// contracts/ast-contract.md): not already inside brackets, and bounded by
// whitespace on entry / whitespace-or-punctuation on exit.
func reconstructHRefs(text string, hrefs []Link, consumed []bool) string {
	if text == "" || len(hrefs) == 0 {
		return text
	}

	type placement struct {
		start, end int
		markup     string
	}

	blocked := existingBracketSpans(text)
	var placements []placement

	overlaps := func(start, end int) bool {
		for _, p := range placements {
			if start < p.end && end > p.start {
				return true
			}
		}
		for _, p := range blocked {
			if start < p.end && end > p.start {
				return true
			}
		}
		return false
	}

	for i, href := range hrefs {
		if consumed[i] {
			continue
		}
		display := href.Alias
		if display == "" {
			display = href.Target
		}
		if display == "" {
			continue
		}

		searchFrom := 0
		for {
			pos := strings.Index(text[searchFrom:], display)
			if pos < 0 {
				break
			}
			start := searchFrom + pos
			end := start + len(display)

			if precededByWhitespace(text, start) && followedByBoundary(text, end) && !overlaps(start, end) {
				placements = append(placements, placement{start: start, end: end, markup: markupFor(href)})
				consumed[i] = true
				break
			}
			searchFrom = start + 1
		}
	}

	if len(placements) == 0 {
		return text
	}

	sort.Slice(placements, func(i, j int) bool { return placements[i].start < placements[j].start })

	var b strings.Builder
	last := 0
	for _, p := range placements {
		b.WriteString(text[last:p.start])
		b.WriteString(p.markup)
		last = p.end
	}
	b.WriteString(text[last:])
	return b.String()
}

func precededByWhitespace(s string, at int) bool {
	if at == 0 {
		return true
	}
	r, _ := utf8.DecodeLastRuneInString(s[:at])
	return unicode.IsSpace(r)
}

func followedByBoundary(s string, at int) bool {
	if at == len(s) {
		return true
	}
	r, _ := utf8.DecodeRuneInString(s[at:])
	return unicode.IsSpace(r) || unicode.IsPunct(r)
}
