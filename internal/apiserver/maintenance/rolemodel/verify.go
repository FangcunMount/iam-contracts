package rolemodel

import "fmt"

// verifyPlan checks the persisted result, not merely the intended writes. It
// runs before committing the receipt and again during cutover verification.
func verifyPlan(s State, p Plan) error {
	in, err := s.Input(nil)
	if err != nil {
		return err
	}
	if len(in.Edges) != 0 {
		return fmt.Errorf("active inheritance remains")
	}
	roles := map[string]bool{}
	for _, r := range in.Roles {
		roles[r.Name] = true
		if _, ok := p.Grants[r.Name]; !ok {
			return fmt.Errorf("unexpected active role after migration: %s", r.Name)
		}
		if (r.Name == Operator || r.Name == Reviewer) && r.Protection != "standard" {
			return fmt.Errorf("new business role must be standard: %s", r.Name)
		}
	}
	grants := map[string][]Permission{}
	for _, g := range in.Grants {
		grants[g.Role] = append(grants[g.Role], g.Permission)
	}
	for name, want := range p.Grants {
		if !roles[name] || !samePermissions(grants[name], want) {
			return fmt.Errorf("persisted role permission mismatch: %s", name)
		}
	}
	want := map[string]bool{}
	for _, m := range p.People {
		for _, name := range m.After {
			want[m.Subject+"/"+name] = true
		}
	}
	for _, a := range in.Assignments {
		key := a.Subject + "/" + a.Role
		if !want[key] {
			return fmt.Errorf("unexpected or duplicate migrated assignment: %s", key)
		}
		delete(want, key)
	}
	if len(want) != 0 {
		return fmt.Errorf("missing %d migrated assignments", len(want))
	}
	for _, r := range in.Resources {
		if r.Key == "iam:authz:collection:role_inheritances" {
			return fmt.Errorf("inheritance management resource is still active")
		}
	}
	return nil
}
