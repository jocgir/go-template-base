# Go template extended

## Overview

This package is intended to extend the default go package `text/template`. Since it is not possible to
access the internals of the original package, the whole source code has been duplicated and is regularly
updated to implement the latest changes in the original go distribution.

Most changes have been done in distinct files to avoid merging conflict with the original library.

By default, there is no difference between this library and the original one except the following exception:

If you register a function that only return error:

```go
t := template.New("test").Funcs(template.FuncMap{
    "my_func": func() error { return fmt.Errorf("bang") },
})
```

That will raise the following error: `can't install method/function "my_func" with only error as result`.

To avoid this error, you will have to register your functions with `ExtraFuncs` method instead of `Funcs`.

## Usage

Instead of importing like this:

```go
package your_package

import (
    "text/template"
)

// Your code
// ...
```

You import it list this:

```go
package your_package

import (
    "github.com/jocgir/template"
)

// Your code
// ...
```

## Why using that implementation instead of the original

So, what are the benefits of this implementation?

### Handling non standard return functions

The original library will fail if you try to register custom functions that have no returns or returns multiple values.

```go
t := template.New("test").Funcs(template.FuncMap{
    "empty":    func() { ... },                            // Will raise: can't install method/function "empty" with 0 results
    "multiple": func() (int, string) { return 0, "Zero" }, // Will raise: can't install method/function "multiple" with 2 results
})
```

Using this implementation will handle these exceptions:

```go
t := template.New("test").ExtraFuncs(template.FuncMap{
    "empty":    func() { ... },
    "multiple": func() (int, string) { return 0, "Zero" },results
})
```

One problem with the original library is that non-compliant custom functions are detected at registration,
but calling non-compliant methods fail at runtime as you can see in that [example](https://github.com/jocgir/template).
