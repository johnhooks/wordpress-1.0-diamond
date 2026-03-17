package permission

import "context"

// FakeChecker is a test double for Checker. Use AllowAll or DenyAll for
// simple cases, or set CanFunc/CanTokenFunc for custom logic.
//
// This lives in the permission package (not _test) so that handler tests
// in other packages can import it.
type FakeChecker struct {
	CanFunc      func(ctx context.Context, userID int64, action Action, object Object) (bool, error)
	CanTokenFunc func(ctx context.Context, token string, action Action, object Object) (bool, error)
}

func (f *FakeChecker) Can(ctx context.Context, userID int64, action Action, object Object) (bool, error) {
	if f.CanFunc != nil {
		return f.CanFunc(ctx, userID, action, object)
	}
	return false, nil
}

func (f *FakeChecker) CanToken(ctx context.Context, token string, action Action, object Object) (bool, error) {
	if f.CanTokenFunc != nil {
		return f.CanTokenFunc(ctx, token, action, object)
	}
	return false, nil
}

// AllowAll returns a Checker that allows every action.
func AllowAll() Checker {
	return &FakeChecker{
		CanFunc:      func(context.Context, int64, Action, Object) (bool, error) { return true, nil },
		CanTokenFunc: func(context.Context, string, Action, Object) (bool, error) { return true, nil },
	}
}

// DenyAll returns a Checker that denies every action.
func DenyAll() Checker {
	return &FakeChecker{
		CanFunc:      func(context.Context, int64, Action, Object) (bool, error) { return false, nil },
		CanTokenFunc: func(context.Context, string, Action, Object) (bool, error) { return false, nil },
	}
}
