package automation

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/thebanri/limoni/core/accessibility"
	"github.com/thebanri/limoni/core/cell"
)

func sensitiveTree() []accessibility.Node {
	return []accessibility.Node{
		// A careless widget author: marked sensitive, but still filled Value in.
		// The gateway must clear it anyway.
		{ID: "password", Role: accessibility.RoleInput, Label: "Password", Value: "hunter2",
			State: accessibility.StateSensitive, Bounds: cell.NewRect(0, 0, 20, 1)},
		{ID: "email", Role: accessibility.RoleInput, Label: "Email", Value: "a@b.c",
			Bounds: cell.NewRect(0, 1, 20, 1)},
		{ID: "form", Role: accessibility.RoleDialog, Label: "Login",
			Children: []accessibility.Node{
				{ID: "nested-secret", Role: accessibility.RoleInput, Value: "nested-hunter2",
					State: accessibility.StateSensitive},
			}},
	}
}

func startWithPolicy(t *testing.T, policy Policy) *Client {
	t.Helper()
	policy.AllowUnverifiedPeers = policy.AllowUnverifiedPeers || !peerVerificationSupported
	server, err := Listen(shortSocketPath(t), WithPolicy(policy))
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	t.Cleanup(func() { _ = server.Close() })
	rec := &recorder{}
	server.Publish(Snapshot{Tree: sensitiveTree(), Screen: "secret screen", Width: 80, Height: 24, Injector: rec.inject})
	client, err := Dial(server.Addr())
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })
	return client
}

// Serialises everything a client can obtain and asserts the secret is absent
// from the bytes, which is the only check that cannot be fooled by a field the
// test forgot to look at.
func assertNeverOnTheWire(t *testing.T, client *Client, secrets ...string) {
	t.Helper()
	tree, err := client.Tree()
	if err != nil {
		t.Fatalf("Tree: %v", err)
	}
	wire, _ := json.Marshal(tree)
	for _, s := range secrets {
		if strings.Contains(string(wire), s) {
			t.Errorf("%q reached the wire: %s", s, wire)
		}
	}
}

func TestDefaultPolicyIsClosed(t *testing.T) {
	client := startWithPolicy(t, Policy{})

	assertNeverOnTheWire(t, client, "hunter2", "a@b.c")

	if _, err := client.Screen(); err == nil {
		t.Error("screen snapshot allowed under the default policy")
	}
	if err := client.Key("enter"); err == nil {
		t.Error("key synthesis allowed under the default policy")
	}
	if err := client.Type("x"); err == nil {
		t.Error("text synthesis allowed under the default policy")
	}
	if err := client.Click(Selector{ID: "email"}); err == nil {
		t.Error("click allowed under the default policy")
	}

	// Structure is still visible, which is what makes observe-only useful.
	nodes, err := client.Find(Selector{Role: "input"})
	if err != nil || len(nodes) != 3 {
		t.Errorf("structure not visible: %d nodes, err %v", len(nodes), err)
	}
}

func TestSensitiveValuesNeverLeaveEvenWhenInputsAreExposed(t *testing.T) {
	client := startWithPolicy(t, Policy{ExposeInputValues: true, ExposeScreen: true, AllowInput: true})

	assertNeverOnTheWire(t, client, "hunter2", "nested-hunter2")

	// The non-sensitive field is exposed when asked for.
	nodes, err := client.Find(Selector{ID: "email"})
	if err != nil || len(nodes) != 1 || nodes[0].Value != "a@b.c" {
		t.Errorf("exposed input value missing: %+v, err %v", nodes, err)
	}
}

// Matching against the raw tree would let a client guess a secret and learn
// from whether anything matched.
func TestSelectorCannotBeUsedAsAnOracle(t *testing.T) {
	client := startWithPolicy(t, Policy{ExposeInputValues: true, AllowInput: true})

	for _, guess := range []string{"hunter2", "nested-hunter2"} {
		nodes, err := client.Find(Selector{Value: guess})
		if err != nil {
			t.Fatalf("Find: %v", err)
		}
		if len(nodes) != 0 {
			t.Errorf("guessing %q matched %d nodes: the value is inferable", guess, len(nodes))
		}
		if err := client.Click(Selector{Value: guess}); err == nil {
			t.Errorf("clicking by guessed value %q succeeded: the value is inferable", guess)
		}
	}
}

func TestHelloReportsPolicy(t *testing.T) {
	client := startWithPolicy(t, Policy{ExposeScreen: true})
	resp, err := client.Do(Request{Op: OpHello})
	if err != nil {
		t.Fatalf("Hello: %v", err)
	}
	if resp.AllowInput || !resp.ExposeScreen || resp.ExposeInputValues {
		t.Errorf("hello policy = input:%v screen:%v values:%v", resp.AllowInput, resp.ExposeScreen, resp.ExposeInputValues)
	}
}

// The published tree belongs to the application; redacting a copy must not
// blank the application's own values.
func TestRedactionDoesNotMutateThePublishedTree(t *testing.T) {
	tree := sensitiveTree()
	_ = Policy{}.redactTree(tree)
	if tree[0].Value != "hunter2" || tree[1].Value != "a@b.c" || tree[2].Children[0].Value != "nested-hunter2" {
		t.Errorf("redaction mutated the source tree: %+v", tree)
	}
}
