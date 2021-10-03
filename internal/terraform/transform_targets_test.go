package terraform

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform/internal/addrs"
)

func TestTargetsTransformer(t *testing.T) {
	mod := testModule(t, "transform-targets-basic")

	g := Graph{Path: addrs.RootModuleInstance}
	{
		tf := &ConfigTransformer{Config: mod}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	{
		transform := &AttachResourceConfigTransformer{Config: mod}
		if err := transform.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	{
		transform := &ReferenceTransformer{}
		if err := transform.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	{
		transform := &TargetsTransformer{
			Targets: []addrs.Targetable{
				addrs.RootModuleInstance.Resource(
					addrs.ManagedResourceMode, "aws_instance", "me",
				),
			},
		}
		if err := transform.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(`
aws_instance.me
  aws_subnet.me
aws_subnet.me
  aws_vpc.me
aws_vpc.me
	`)
	if actual != expected {
		t.Fatalf("bad:\n\nexpected:\n%s\n\ngot:\n%s\n", expected, actual)
	}
}

/* Graph for nothing:
aws_instance.me
          aws_subnet.me
        aws_instance.notme
        aws_instance.notmeeither
          aws_instance.me
        aws_subnet.me
          aws_vpc.me
        aws_subnet.notme
        aws_vpc.me
        aws_vpc.notme */

//issue: it targets aws_vpc.me
func TestExcludeTargetsTransformer(t *testing.T) {
	mod := testModule(t, "transform-targets-basic")

	g := Graph{Path: addrs.RootModuleInstance}
	{
		tf := &ConfigTransformer{Config: mod}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	{
		transform := &AttachResourceConfigTransformer{Config: mod}
		if err := transform.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	{
		transform := &ReferenceTransformer{}
		if err := transform.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	{
		transform := &TargetsTransformer{
			ExcludeTargets: []addrs.Targetable{
				addrs.RootModuleInstance.Resource(
					addrs.ManagedResourceMode, "aws_subnet", "me",
				),
			},
		}
		if err := transform.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}
	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(`
aws_instance.me
  aws_subnet.me
aws_subnet.me
  aws_vpc.me
aws_vpc.me
	`)
	if actual != expected {
		t.Fatalf("bad:\n\nexpected:\n%s\n\ngot:\n%s\n", expected, actual)
	}
}
func TestTargetsTransformer_downstream(t *testing.T) {
	mod := testModule(t, "transform-targets-downstream")

	g := Graph{Path: addrs.RootModuleInstance}
	{
		transform := &ConfigTransformer{Config: mod}
		if err := transform.Transform(&g); err != nil {
			t.Fatalf("%T failed: %s", transform, err)
		}
	}

	{
		transform := &AttachResourceConfigTransformer{Config: mod}
		if err := transform.Transform(&g); err != nil {
			t.Fatalf("%T failed: %s", transform, err)
		}
	}

	{
		transform := &AttachResourceConfigTransformer{Config: mod}
		if err := transform.Transform(&g); err != nil {
			t.Fatalf("%T failed: %s", transform, err)
		}
	}

	{
		transform := &OutputTransformer{Config: mod}
		if err := transform.Transform(&g); err != nil {
			t.Fatalf("%T failed: %s", transform, err)
		}
	}

	{
		transform := &ReferenceTransformer{}
		if err := transform.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	{
		transform := &TargetsTransformer{
			Targets: []addrs.Targetable{
				addrs.RootModuleInstance.
					Child("child", addrs.NoKey).
					Child("grandchild", addrs.NoKey).
					Resource(
						addrs.ManagedResourceMode, "aws_instance", "foo",
					),
			},
		}
		if err := transform.Transform(&g); err != nil {
			t.Fatalf("%T failed: %s", transform, err)
		}
	}

	{
		transform := &TargetsTransformer{
			Targets: []addrs.Targetable{
				addrs.RootModuleInstance.
					Child("child", addrs.NoKey).
					Child("grandchild", addrs.NoKey).
					Resource(
						addrs.ManagedResourceMode, "aws_instance", "foo",
					),
			},
		}
		if err := transform.Transform(&g); err != nil {
			t.Fatalf("%T failed: %s", transform, err)
		}
	}

	actual := strings.TrimSpace(g.String())
	// Even though we only asked to target the grandchild resource, all of the
	// outputs that descend from it are also targeted.
	expected := strings.TrimSpace(`
module.child.module.grandchild.aws_instance.foo
module.child.module.grandchild.output.id (expand)
  module.child.module.grandchild.aws_instance.foo
module.child.output.grandchild_id (expand)
  module.child.module.grandchild.output.id (expand)
output.grandchild_id
  module.child.output.grandchild_id (expand)
	`)
	if actual != expected {
		t.Fatalf("bad:\n\nexpected:\n%s\n\ngot:\n%s\n", expected, actual)
	}
}

/* Graph w/o Transformations:
aws_instance.foo
        module.child.aws_instance.foo
        /module.child.module.grandchild.aws_instance.foo
        /module.child.module.grandchild.output.id (expand)
          /module.child.module.grandchild.aws_instance.foo
        /module.child.output.grandchild_id (expand)
          /module.child.module.grandchild.output.id (expand)
        module.child.output.id (expand)
          module.child.aws_instance.foo
        output.child_id
          module.child.output.id (expand)
        /output.grandchild_id
          /module.child.output.grandchild_id (expand)
        output.root_id
          aws_instance.foo
*/

func TestExcludeTargetsTransformer_downstream(t *testing.T) {
	mod := testModule(t, "transform-targets-downstream")

	g := Graph{Path: addrs.RootModuleInstance}
	{
		transform := &ConfigTransformer{Config: mod}
		if err := transform.Transform(&g); err != nil {
			t.Fatalf("%T failed: %s", transform, err)
		}
	}

	{
		transform := &AttachResourceConfigTransformer{Config: mod}
		if err := transform.Transform(&g); err != nil {
			t.Fatalf("%T failed: %s", transform, err)
		}
	}

	{
		transform := &AttachResourceConfigTransformer{Config: mod}
		if err := transform.Transform(&g); err != nil {
			t.Fatalf("%T failed: %s", transform, err)
		}
	}

	{
		transform := &OutputTransformer{Config: mod}
		if err := transform.Transform(&g); err != nil {
			t.Fatalf("%T failed: %s", transform, err)
		}
	}

	{
		transform := &ReferenceTransformer{}
		if err := transform.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	{
		transform := &TargetsTransformer{
			ExcludeTargets: []addrs.Targetable{
				addrs.RootModuleInstance.
					Child("child", addrs.NoKey).
					Child("grandchild", addrs.NoKey).
					Resource(
						addrs.ManagedResourceMode, "aws_instance", "foo",
					),
			},
		}
		if err := transform.Transform(&g); err != nil {
			t.Fatalf("%T failed: %s", transform, err)
		}
	}

	actual := strings.TrimSpace(g.String())
	// Even though we only asked to target the grandchild resource, all of the
	// outputs that descend from it are also targeted.
	expected := strings.TrimSpace(`
aws_instance.foo
module.child.aws_instance.foo
module.child.output.id (expand)
	module.child.aws_instance.foo
output.child_id
	module.child.output.id (expand)
output.root_id
	aws_instance.foo
	`)
	if actual != expected {
		t.Fatalf("bad:\n\nexpected:\n%s\n\ngot:\n%s\n", expected, actual)
	}
}

// This tests the TargetsTransformer targeting a whole module,
// rather than a resource within a module instance.
func TestTargetsTransformer_wholeModule(t *testing.T) {
	mod := testModule(t, "transform-targets-downstream")

	g := Graph{Path: addrs.RootModuleInstance}
	{
		transform := &ConfigTransformer{Config: mod}
		if err := transform.Transform(&g); err != nil {
			t.Fatalf("%T failed: %s", transform, err)
		}
	}

	{
		transform := &AttachResourceConfigTransformer{Config: mod}
		if err := transform.Transform(&g); err != nil {
			t.Fatalf("%T failed: %s", transform, err)
		}
	}

	{
		transform := &AttachResourceConfigTransformer{Config: mod}
		if err := transform.Transform(&g); err != nil {
			t.Fatalf("%T failed: %s", transform, err)
		}
	}

	{
		transform := &OutputTransformer{Config: mod}
		if err := transform.Transform(&g); err != nil {
			t.Fatalf("%T failed: %s", transform, err)
		}
	}

	{
		transform := &ReferenceTransformer{}
		if err := transform.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	{
		transform := &TargetsTransformer{
			Targets: []addrs.Targetable{
				addrs.RootModule.
					Child("child").
					Child("grandchild"),
			},
		}
		if err := transform.Transform(&g); err != nil {
			t.Fatalf("%T failed: %s", transform, err)
		}
	}

	actual := strings.TrimSpace(g.String())
	// Even though we only asked to target the grandchild module, all of the
	// outputs that descend from it are also targeted.
	expected := strings.TrimSpace(`
<<<<<<< HEAD
aws_instance.foo
module.child.aws_instance.foo
module.child.output.id (expand)
	module.child.aws_instance.foo
output.child_id
	module.child.output.id (expand)
output.root_id
	aws_instance.foo
||||||| parent of c77a9c5b73 (Revert change to original test)
aws_instance.foo
module.child.aws_instance.foo
module.child.output.id (expand)
  module.child.aws_instance.foo
output.child_id
  module.child.output.id (expand)
output.root_id
  aws_instance.foo
=======
module.child.module.grandchild.aws_instance.foo
module.child.module.grandchild.output.id (expand)
  module.child.module.grandchild.aws_instance.foo
module.child.output.grandchild_id (expand)
  module.child.module.grandchild.output.id (expand)
output.grandchild_id
  module.child.output.grandchild_id (expand)
>>>>>>> c77a9c5b73 (Revert change to original test)
	`)
	if actual != expected {
		t.Fatalf("bad:\n\nexpected:\n%s\n\ngot:\n%s\n", expected, actual)
	}
}

/* Graph w/o Transform:
		aws_instance.foo
        module.child.aws_instance.foo
        module.child.module.grandchild.aws_instance.foo
        module.child.module.grandchild.output.id (expand)
          module.child.module.grandchild.aws_instance.foo
        module.child.output.grandchild_id (expand)
          module.child.module.grandchild.output.id (expand)
        module.child.output.id (expand)
          module.child.aws_instance.foo
        output.child_id
          module.child.output.id (expand)
        output.grandchild_id
          module.child.output.grandchild_id (expand)
        output.root_id
          aws_instance.foo
*/

func TestExcludeTargetsTransformer_wholeModule(t *testing.T) {
	mod := testModule(t, "transform-targets-downstream")

	g := Graph{Path: addrs.RootModuleInstance}
	{
		transform := &ConfigTransformer{Config: mod}
		if err := transform.Transform(&g); err != nil {
			t.Fatalf("%T failed: %s", transform, err)
		}
	}

	{
		transform := &AttachResourceConfigTransformer{Config: mod}
		if err := transform.Transform(&g); err != nil {
			t.Fatalf("%T failed: %s", transform, err)
		}
	}

	{
		transform := &AttachResourceConfigTransformer{Config: mod}
		if err := transform.Transform(&g); err != nil {
			t.Fatalf("%T failed: %s", transform, err)
		}
	}

	{
		transform := &OutputTransformer{Config: mod}
		if err := transform.Transform(&g); err != nil {
			t.Fatalf("%T failed: %s", transform, err)
		}
	}

	{
		transform := &ReferenceTransformer{}
		if err := transform.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	{
		transform := &TargetsTransformer{
			ExcludeTargets: []addrs.Targetable{
				addrs.RootModule.
					Child("child").
					Child("grandchild"),
			},
		}
		if err := transform.Transform(&g); err != nil {
			t.Fatalf("%T failed: %s", transform, err)
		}
	}

	actual := strings.TrimSpace(g.String())
	// Even though we only asked to exclude the grandchild module, all of the
	// outputs that descend from it are also targeted.
	expected := strings.TrimSpace(`
aws_instance.foo
module.child.aws_instance.foo
module.child.output.id (expand)
	module.child.aws_instance.foo
output.child_id
	module.child.output.id (expand)
output.root_id
aws_instance.foo
	`)
	if actual != expected {
		t.Fatalf("bad:\n\nexpected:\n%s\n\ngot:\n%s\n", expected, actual)
	}
}
