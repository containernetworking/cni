# libcni

`libcni` is a library that parses and loads CNI configurations as defined in the [CNI spec](../SPEC.md).

It is designed to be used by runtimes for this purpose, and is kept in sync with the CNI spec as a reference library implementation so that runtimes do not have to build and maintain their own implementations of the CNI spec, or construct their own parsing and loading logic.

It is not required to use this library to be compliant with the [CNI spec](../SPEC.md).

While the [CNI spec](../SPEC.md) only dictates the API and types, and does not dictate operational concerns or how or where from configuration is loaded, `libcni` is an opinionated, file-based implementation, and primarily loads and validates CNI spec-compliant configuration files from disk.

`libcni` is versioned independently from the CNI spec.
