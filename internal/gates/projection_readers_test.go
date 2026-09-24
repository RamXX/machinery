package gates

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/RamXX/machinery/internal/checker"
	"github.com/RamXX/machinery/internal/datalog"
)

// bundledDesigns lists every example design from examples/inventory.tsv.
func bundledDesigns(t *testing.T) []string {
	t.Helper()
	f, err := os.Open(filepath.Join("..", "..", "examples", "inventory.tsv"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var out []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, "#") || strings.TrimSpace(line) == "" {
			continue
		}
		out = append(out, strings.SplitN(line, "\t", 2)[0])
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	if len(out) == 0 {
		t.Fatal("examples/inventory.tsv lists no designs")
	}
	return out
}

// v1LayerNames are the layers whose JSON also needs the v1 model reader.
var v1LayerNames = map[string]bool{"model": true, "invariants": true, "relationships": true}

// allLayersManifest is a manifest that includes every layer the facts carry,
// or every non-v1 layer when withV1 is false.
func allLayersManifest(t *testing.T, facts *checker.DesignFacts, withV1 bool) *checker.Manifest {
	t.Helper()
	man := &checker.Manifest{}
	man.Checker.ID = "roundtrip"
	for _, layer := range facts.Layers() {
		if withV1 || !v1LayerNames[layer] {
			man.Projection.Include = append(man.Projection.Include, layer)
		}
	}
	return man
}

// countProgram declares every relation of relations.txt as .input and
// outputs one count per relation.
func countProgram(t *testing.T, index string) (string, []string) {
	t.Helper()
	var b strings.Builder
	var names []string
	for _, line := range strings.Split(strings.TrimSuffix(index, "\n"), "\n") {
		fields := strings.Split(line, "\t")
		if len(fields) != 4 {
			t.Fatalf("relations.txt line %q must carry name, arity, layer and columns", line)
		}
		name, columns := fields[0], strings.Split(fields[3], ",")
		arity, err := strconv.Atoi(fields[1])
		if err != nil || arity != len(columns) {
			t.Fatalf("relations.txt line %q: arity disagrees with the columns", line)
		}
		var decl, wild []string
		for _, c := range columns {
			decl = append(decl, c+":symbol")
			wild = append(wild, "_")
		}
		fmt.Fprintf(&b, ".decl %s(%s)\n.input %s\n", name, strings.Join(decl, ", "), name)
		fmt.Fprintf(&b, ".decl n_%s(n:number)\n.output n_%s\n", name, name)
		fmt.Fprintf(&b, "n_%s(N) :- N = count : { %s(%s) }.\n", name, name, strings.Join(wild, ", "))
		names = append(names, name)
	}
	return b.String(), names
}

// jsonRelationCounts counts rows per relation in a rendered v2 projection,
// reading the JSON generically rather than through the Go types.
func jsonRelationCounts(t *testing.T, rendered []byte) map[string]int {
	t.Helper()
	var doc struct {
		Layers map[string]map[string][]json.RawMessage `json:"layers"`
	}
	if err := json.Unmarshal(rendered, &doc); err != nil {
		t.Fatal(err)
	}
	out := map[string]int{}
	for _, rels := range doc.Layers {
		for name, rows := range rels {
			out[name] = len(rows)
		}
	}
	return out
}

func TestFactsRoundTripAgreesWithJSONForEveryExample(t *testing.T) {
	for _, rel := range bundledDesigns(t) {
		t.Run(rel, func(t *testing.T) {
			design := filepath.Join("..", "..", filepath.FromSlash(rel))
			facts, err := LoadDesignFacts(design)
			if err != nil {
				t.Fatal(err)
			}
			dir := filepath.Join(t.TempDir(), "facts")
			if err := checker.WriteFactsDir(dir, facts); err != nil {
				t.Fatal(err)
			}
			index, err := os.ReadFile(filepath.Join(dir, checker.RelationsIndexFile))
			if err != nil {
				t.Fatal(err)
			}
			src, names := countProgram(t, string(index))
			prog, err := datalog.Parse(src, "count.dl")
			if err != nil {
				t.Fatal(err)
			}
			in, err := datalog.ReadFacts(dir)
			if err != nil {
				t.Fatal(err)
			}
			res, err := prog.Run(in, datalog.Options{})
			if err != nil {
				t.Fatal(err)
			}
			// The v1 model reader refuses a cardinality Modelith accepts (n:n);
			// such a design cannot carry the v1 layers in JSON at all, so its
			// round trip covers every other layer and says so.
			model, modelErr := checker.LoadModel(checker.ModelPaths(design)[0])
			withV1 := modelErr == nil
			if !withV1 {
				t.Logf("%s: v1 model reader refuses the model (%v); JSON compared without the v1 layers", rel, modelErr)
			}
			proj, err := checker.GenerateWithFacts(model, facts, allLayersManifest(t, facts, withV1), "sha256:"+strings.Repeat("0", 64), "test")
			if err != nil {
				t.Fatal(err)
			}
			rendered, err := proj.Render()
			if err != nil {
				t.Fatal(err)
			}
			jsonCounts := jsonRelationCounts(t, rendered)
			var report []string
			compared := 0
			for _, name := range names {
				out := res.Output("n_" + name)
				if len(out) != 1 {
					t.Fatalf("count of %s yielded %d rows", name, len(out))
				}
				n, err := strconv.Atoi(out[0][0])
				if err != nil {
					t.Fatal(err)
				}
				report = append(report, fmt.Sprintf("%s=%d", name, n))
				want, ok := jsonCounts[name]
				if !ok {
					if !withV1 && v1LayerNames[relationLayer(name)] {
						continue
					}
					t.Fatalf("relation %s is in the facts directory but not in the JSON projection", name)
				}
				compared++
				if n != want {
					t.Errorf("relation %s: facts count %d, JSON count %d", name, n, want)
				}
			}
			if compared != len(jsonCounts) {
				t.Fatalf("JSON carries %d relations, %d were compared", len(jsonCounts), compared)
			}
			t.Logf("%s: %s", rel, strings.Join(report, " "))
		})
	}
}

func relationLayer(name string) string {
	for _, spec := range checker.RelationCatalog() {
		if spec.Name == name {
			return spec.Layer
		}
	}
	return ""
}

// requireRelationCount asserts a relation's tuple count in a fact set.
func requireRelationCount(t *testing.T, facts *checker.DesignFacts, relation string, atLeast int) {
	t.Helper()
	if got := len(facts.Rows(relation)); got < atLeast {
		t.Fatalf("relation %s has %d rows, want at least %d", relation, got, atLeast)
	}
}

func TestStageOneDeclarationsReachTheFacts(t *testing.T) {
	fulfillment, err := LoadDesignFacts(filepath.Join("..", "..", "examples", "fulfillment", "design"))
	if err != nil {
		t.Fatal(err)
	}
	for _, relation := range []string{"unit_writes", "unit_uses", "unit_carries"} {
		requireRelationCount(t, fulfillment, relation, 1)
	}
	var writes []string
	for _, row := range fulfillment.Rows("unit_writes") {
		writes = append(writes, strings.Join(row.Values, " "))
	}
	if !contains(writes, "Order.persistOrder Order.status") {
		t.Fatalf("persistOrder WRITES{Order.status} must project: %v", writes)
	}
	goCRM, err := LoadDesignFacts(filepath.Join("..", "..", "examples", "go-crm", "design"))
	if err != nil {
		t.Fatal(err)
	}
	requireRelationCount(t, goCRM, "supersedes", 1)
	requireRelationCount(t, goCRM, "type_owner", 1)
	row := goCRM.Rows("supersedes")[0]
	if strings.Join(row.Values, " ") != "Deal LegacyDeal" || row.StableID != "type:Deal" || row.Source.Path != "ARCHITECTURE.md" {
		t.Fatalf("SUPERSEDES{type:LegacyDeal} on the Deal row must project as supersedes(Deal, LegacyDeal): %+v", row)
	}
}

func contains(xs []string, want string) bool {
	for _, x := range xs {
		if x == want {
			return true
		}
	}
	return false
}

// factsBytes renders a fact set's files as one comparable blob.
func factsBytes(t *testing.T, facts *checker.DesignFacts) []byte {
	t.Helper()
	files, err := facts.FactsFiles()
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	var b bytes.Buffer
	for _, name := range names {
		fmt.Fprintf(&b, "== %s\n%s", name, files[name])
	}
	return b.Bytes()
}
