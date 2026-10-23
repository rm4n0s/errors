# github.com/rm4n0s/errors — Guide for AI coding assistants

Read this whole file before writing code that uses this package. It is short on purpose. Every rule below is
there because models have broken it before.

## 1. Mental model

- An error is identified by its **Tag** (a string such as `"UserAlreadyExists"`), not by its message.
  A Tag names one thing a function **can't** do.
- `New`, `NewErr` and `NewErrf` record the **call stack at the line where they are called**.
- `Route()` turns that stack into a path string such as `UserService.CreateUser->UserRepository.Save.UserAlreadyExists`.
- Tests assert **which path** produced the error with `HasRoute(...)`, not only that an error happened.
- `ToJson()` / `FromJson()` let a logged error be replayed in a test, and the route survives the round trip.

So: **create the error once, where the failure happens, and return it unchanged all the way up.**

## 2. Complete API

```go
import "github.com/rm4n0s/errors"

// Constructors. Each one captures the call stack at the call site.
errors.New(tag, message string, metadata ...any) *Error   // metadata = alternating key, value, key, value...
errors.NewErr(tag string, err error) *Error                // wraps a foreign error; Message = err.Error(); err must NOT be nil
errors.NewErrf(tag, format string, a ...any) *Error        // like fmt.Errorf (supports %w); result is OriginalErr

// Conversion
errors.FromError(err error) (*Error, bool)                 // plain type assertion; does NOT unwrap
errors.FromJson(errj ErrorJson) (*Error, error)            // rebuild from a logged ErrorJson; error is always nil

// Re-exports of the standard library, so you never need to import both packages
errors.Is, errors.As, errors.Unwrap

// Methods on *Error (all setters return the same *Error, so they chain)
(*Error).SetMetadata(kv ...any) *Error       // MERGES into existing metadata; later keys overwrite earlier ones
(*Error).RemoveMetadata(keys ...string) *Error // deletes those keys if present; missing keys are ignored
(*Error).SetMessage(msg string) *Error       // replaces Message
(*Error).Route() string                      // "A->B.C->D.Tag"
(*Error).HasRoute(sub string) bool           // substring match against the route
(*Error).ToJson() (*ErrorJson, error)        // error is always nil
(*Error).StackFrames(all bool) []StackFrame
(*Error).StackTrace() string                 // human-readable trace with file:line
(*Error).Is(target error) bool               // true when target is *Error with the same Tag
(*Error).Error() string                      // returns Message
(*Error).Unwrap() error                      // returns OriginalErr

// ErrorJson (what you log and replay)
type ErrorJson struct { Tag, Message, OriginalErr string; Metadata map[string]any; StackFrames []StackFrame }
(*ErrorJson).Route() string
(*ErrorJson).HasRoute(sub string) bool
```

Exported fields of `Error`: `Tag string`, `Message string`, `Metadata map[string]any`, `OriginalErr error`.

## 3. Rules

### R1. `errors.New` takes a tag AND a message. It is not the standard library's `errors.New`.

```go
// WRONG: Tag becomes "user not found" and Message is empty
return errors.New("user not found")

// RIGHT
return errors.New("UserNotFound", "user not found", "userID", id)
```

### R2. Put all metadata in ONE call, as alternating key/value pairs.

```go
// WRONG: verbose, one call per key
return errors.New("InvalidSize", "size out of range").SetMetadata("w", w).SetMetadata("h", h)

// RIGHT
return errors.New("InvalidSize", "size out of range", "w", w, "h", h)
```

`NewErr` and `NewErrf` take no metadata argument, so chain exactly **one** `SetMetadata` call with all the pairs:

```go
return errors.NewErr("DBInsertFailed", err).SetMetadata("table", "users", "email", u.Email)
```

`SetMetadata` merges, so a second call adds keys instead of losing the first ones. Use a later call only when
the data really becomes available later, for example in a different function. Keys must be `string`. A
non-string in a key position is stored under `"arg<index>"`, and a trailing key with no value is stored as
`nil`. Both are bugs.

### R3. When a call returns an error, what you do depends on who returned it.

Every error is tagged exactly once, at the place where it first appears in your code. After that it travels
up as the same value.

| The `err` came from…                                                        | Do this                          |
|-----------------------------------------------------------------------------|----------------------------------|
| A function **in your project** (it already returns an `*errors.Error`)      | `return err`                     |
| **Outside your project**: standard library, database driver, HTTP client, any third-party package | `return errors.NewErr("Tag", err)` |

**Case A: the error comes from your own code. Pass it up untouched.**

```go
func (s *UserService) CreateUser(email string) error {
    if err := s.repo.Save(u); err != nil { // repo is your code, so err is already tagged
        return err
    }
    return nil
}
```

Do **not** do either of these in Case A:

- `return fmt.Errorf("create user: %w", err)` hides the `*Error`, so `errors.FromError(err)` returns
  `ok=false` and tests can no longer read the Tag, Metadata or Route.
- `return errors.NewErr("CreateUserFailed", err)` creates a new error whose Tag is `CreateUserFailed` and
  whose Route starts again from here. The path down to `Save.UserAlreadyExists` disappears from `Route()`.

You don't need to add context by wrapping. The route already shows that `CreateUser` called `Save`.

**Case B: the error comes from outside your project. Tag it here, once.**

```go
row, err := db.QueryRow(...)
if err != nil {
    return errors.NewErr("UserQueryFailed", err).SetMetadata("userID", id)
}
```

`errors.Is(err, sql.ErrNoRows)` still works on the result because `Unwrap` returns the original error.
`NewErr(tag, nil)` panics, so only call it inside `if err != nil`.

When there is no underlying error and your code itself detects the failure, use `New`:

```go
if n > max {
    return errors.New("TooManyItems", "too many items", "count", n, "max", max)
}
```

### R4. Call the constructor in the function that fails, not in a helper.

The route is captured where `New` / `NewErr` / `NewErrf` runs. Every function between that line and the
caller appears in the route.

```go
// WRONG: route becomes "...->Save->newDuplicateErr.UserAlreadyExists"
func newDuplicateErr(email string) *errors.Error {
    return errors.New("UserAlreadyExists", "duplicate email", "email", email)
}

// RIGHT: call errors.New directly inside Save
```

Anonymous functions also appear in the route, as `Outer.func1`. For example, an error created inside a
closure in `Run` gives `Run->Run.func1.Tag`.

### R5. Declare tags as constants next to the code that produces them, and match on tags, not messages.

```go
package repository

const (
    TagUserAlreadyExists = "UserAlreadyExists"
    TagUserNotFound      = "UserNotFound"
)
```

To compare, use either the tag or a sentinel with `errors.Is`. `(*Error).Is` matches by Tag only:

```go
if e, ok := errors.FromError(err); ok && e.Tag == repository.TagUserNotFound { ... }

var ErrUserNotFound = &errors.Error{Tag: repository.TagUserNotFound}
if errors.Is(err, ErrUserNotFound) { ... }
```

Never branch on `err.Error()` or `Message`. Messages are for humans and may change.

### R6. Import only this package.

It re-exports `Is`, `As` and `Unwrap`. If you really need something it does not export (for example
`errors.Join`), import the standard library under an alias: `stdErrors "errors"`.

## 4. Testing error paths (the reason this package exists)

```go
func TestCreateUser_RejectsDuplicateEmail(t *testing.T) {
    svc := service.NewUserService(repository.NewFake(withExistingUser("a@b.c")))

    err := svc.CreateUser("a@b.c")

    appErr, ok := errors.FromError(err)
    if !ok {
        t.Fatalf("expected *errors.Error, got %T: %v", err, err)
    }
    if !appErr.HasRoute("UserService.CreateUser->UserRepository.Save.UserAlreadyExists") {
        t.Errorf("unexpected failure path: %s", appErr.Route())
    }
    if appErr.Metadata["email"] != "a@b.c" {
        t.Errorf("metadata = %v", appErr.Metadata)
    }
}
```

Route format, exactly:

- Frames run from outermost to innermost, joined by `->`.
- A method frame is `StructName.FuncName` (pointer `*` removed). A function frame is `FuncName`.
- **Package names never appear.**
- The last frame ends with `.` + Tag.
- `Route()` stops at the function that called `Route()`, so in a test it starts just below the test function.
- `HasRoute(s)` is a substring check. Pass the path from the entry function you call down to `.Tag`.
  Include the Tag so that the assertion also checks which error happened.
- When an assertion fails, print `appErr.Route()` to see the actual path.

## 5. Logging and replaying

```go
if e, ok := errors.FromError(err); ok {
    j, _ := e.ToJson()
    b, _ := json.Marshal(j)
    log.Println(string(b))
}
```

```go
func TestReproduceIssue482(t *testing.T) {
    var logged errors.ErrorJson
    if err := json.Unmarshal([]byte(rawLogEntry), &logged); err != nil {
        t.Fatal(err)
    }
    email, _ := logged.Metadata["email"].(string) // Metadata values are `any`, so assert the type

    err := service.NewUserService(productionLikeRepo).CreateUser(email)

    appErr, ok := errors.FromError(err)
    if !ok || appErr.Tag != logged.Tag || !appErr.HasRoute(logged.Route()) {
        t.Fatalf("not reproduced: want %s, got %v", logged.Route(), err)
    }
}
```

After `json.Unmarshal`, numeric metadata values are `float64`, not `int`. Keep metadata values
JSON-serializable: strings, numbers, bools, or simple structs.

## 6. Checklist before you finish

- [ ] Every `errors.New` call has a tag **and** a message.
- [ ] Metadata is passed in one call as `"key", value` pairs, with no chains of `SetMetadata`.
- [ ] Errors from your own project's functions are passed up with `return err`, never wrapped with `fmt.Errorf` or re-tagged with `NewErr`.
- [ ] Errors from outside your project (standard library, drivers, third-party packages) are tagged once, with `NewErr`, where they first appear.
- [ ] Constructors are called inside the failing function, not inside a helper.
- [ ] Tags are constants. Code branches on Tag or `errors.Is`, never on the message.
- [ ] Each tagged failure has a test that asserts its path with `HasRoute("...->...Tag")`.
