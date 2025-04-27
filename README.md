# Gatekeeper

A lightweight, in-memory resource locking package to prevent concurrent access to resources.

## Usage

### Direct Usage

```go
// Lock a user resource
if !gatekeeper.TryLock(gatekeeper.ResourceUser, "user123") {
    return errors.New("user is already being processed")
}
defer gatekeeper.ReleaseLock(gatekeeper.ResourceUser, "user123")

// Process the user...
```

### Middleware Usage

```go
// Create a middleware that locks user resources
userLock := gatekeeper.Middleware(gatekeeper.ResourceUser, func(c *fiber.Ctx) string {
    return utils.GetUserId(c)
})

// Apply middleware to routes
app.Post("/users", userLock, controllers.CreateUser)
```

### Configuration

```go
// Use default settings (no configuration needed)
gatekeeper.Setup()

// Or customize specific options
gatekeeper.Setup(
    gatekeeper.WithLockTimeout(10 * time.Second),
    gatekeeper.WithErrorMessage("Resource busy"),
)
```

Available options:
- `WithLockTimeout(duration)` - How long locks remain active
- `WithEnabled(bool)` - Toggle locking on/off
- `WithErrorStatus(int)` - HTTP status for locked resources
- `WithErrorMessage(string)` - Error message
- `WithErrorCode(string)` - Error code

### Custom Resource Types

```go
// Define your own resource types
const ResourceCart = "cart"

// Use with gatekeeper
if !gatekeeper.TryLock(ResourceCart, cartId) {
    return errors.New("cart is locked")
}
```

## Note

This is an in-memory locking mechanism for single-instance applications. For distributed locking, consider solutions like Redis locks. 