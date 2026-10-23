# errors

## Why

This package provides stack-trace-aware errors, similar to other error-handling libraries — but with one key difference: the Route() and HasRoute() methods let you write unit tests that verify not just that an error occurred, but that it occurred along a specific execution path. <br/>
Combined with test-driven development, this allows you to design a monolithic application by defining, for each function, the errors it should raise — effectively driving your software's architecture through its error paths. <br/>
You can read more about the problem this package addresses in [Error Handling Challenge](https://rm4n0s.github.io/posts/3-error-handling-challenge/), and about the underlying theory of architecting software around its errors in [Can't Driven Development](https://rm4n0s.github.io/posts/6-cant-driven-development/). <br/>
Errors log themselves through log/slog, and a ToJson() method serializes an error to JSON. This lets you take an error captured in your logs and replay it in an integration test, making it straightforward to reproduce and fix the underlying bug. <br/>

## Usage

> AI coding assistants: read [AI_GUIDE.md](AI_GUIDE.md) before using this package.

### Define errors close to where they happen

Give each function a `Tag` for the errors it can produce. `New` captures the call stack automatically.

```go
package repository

import "github.com/rm4n0s/errors"

func (r *UserRepository) Save(user *User) error {
    if exists, _ := r.db.EmailExists(user.Email); exists {
        return errors.New("UserAlreadyExists", "a user with this email already exists", "email", user.Email)
    }
    return r.db.Insert(user)
}
```

```go
package service

func (s *UserService) CreateUser(email string) error {
    user := &User{Email: email}
    if err := s.repo.Save(user); err != nil {
        return err // bubble up untouched — Tag, Metadata and the stack stay intact
    }
    return nil
}
```

### Test that an error came from a specific path

This is the part other error packages don't give you: with this you don't test only a failure but also the execution path followed to the specific failure.

```go
func TestCreateUser_RejectsDuplicateEmail(t *testing.T) {
    repo := repository.NewFake(withExistingUser("[email protected]"))
    svc := service.NewUserService(repo)

    err := svc.CreateUser("[email protected]")

    appErr, ok := errors.FromError(err)
    if !ok {
        t.Fatalf("expected *errors.Error, got %T", err)
    }

    // Not just "an error happened" — it happened in Save, called from CreateUser.
    if !appErr.HasRoute("UserService.CreateUser->UserRepository.Save.UserAlreadyExists") {
        t.Errorf("unexpected failure path: %s", appErr.Route())
    }
}
```
This may not add much value for microservices, where only two or three functions call into each other. But for monolithic applications with long chains of
conditional logic — as is common in ERP-style systems — it becomes crucial.

### Reproduce a production bug from your logs

`*errors.Error` implements `slog.LogValuer`, so logging it with `log/slog` writes the tag, message, route, metadata and stack frames.
The keys match the fields of `ErrorJson`, so the logged object can be unmarshaled straight back into it.

```go
logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
logger.Error("create user failed", "err", err)
// {"time":"...","level":"ERROR","msg":"create user failed","err":{"Tag":"UserAlreadyExists","Message":"...",
//  "Route":"UserService.CreateUser->UserRepository.Save.UserAlreadyExists","Metadata":{"email":"..."},"StackFrames":[...]}}
```

```go
func TestReproduceIssue482(t *testing.T) {
    var entry struct {
        Err errors.ErrorJson `json:"err"`
    }
    if err := json.Unmarshal([]byte(rawLogEntry), &entry); err != nil { // pasted from the log
        t.Fatal(err)
    }
    logged := entry.Err
    t.Logf("original failure path: %s", logged.Route())

    email, _ := logged.Metadata["email"].(string)
    err := service.NewUserService(productionLikeRepo).CreateUser(email)

    appErr, ok := errors.FromError(err)
    if !ok || appErr.Tag != logged.Tag || !appErr.HasRoute(logged.Route()) {
        t.Fatalf("bug not reproduced: want %s, got %v", logged.Route(), err)
    }
}
```

Without slog, use `appErr.ToJson()` and marshal the result yourself, and `errors.FromJson(logged)` to turn it back into an `*errors.Error`.

### Print the stack trace

`fmt.Printf("%+v", err)` prints the message followed by the stack trace. `%v` and `%s` print only the message.
