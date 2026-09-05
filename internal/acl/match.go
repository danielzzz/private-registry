package acl

import "path"

func Match(pattern, name string) bool {
	ok, err := path.Match(pattern, name)
	return err == nil && ok
}
