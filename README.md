# unison-fsmonitor

A filesystem monitoring adapter for the [Unison] file synchronizer with macOS
support.

## Installation

```
go get -u github.com/benesch/unison-fswatcher
```

## Details

In `-repeat watch` mode, Unison can automatically re-sync whenever it notices
that a file has changed. This mode relies on an adapter program,
`unison-fsmonitor`, that provides the system-specific glue for filesystem
notifications. Unison ships with an adapter for Linux and Windows, but not for
macOS.

This `unison-fsmonitor` adapter program is written in Go and supports macOS.
It should, in principle, support other operating systems—the underlying
filesystem notification library supports most popular operating systems—but
it has only been tested on macOS.

Notably, this adapter handles following symlinks better than the other available
macOS adapters.

For the curious, the adapter protocol is documented in the [fswatch.ml
file][protocol] in Unison's source code.

## License

unison-fsmonitor is freely available under the terms of the [MIT
license](LICENSE).

## Related projects

There are several similar projects that provide a `unison-fsmonitor` adapter for
macOS.

* [autozimu/unison-fsmonitor](https://github.com/autozimu/unison-fsmonitor)
* [caleb/unison-fsmonitor-macos](https://github.com/hnsl/unox)
* [hnsl/unox](https://github.com/hnsl/unox)

[Unison]: https://www.cis.upenn.edu/~bcpierce/unison/
[protocol]: https://github.com/bcpierce00/unison/blob/d04ef5b5a368bc092487bf49432bc8a29c628687/src/fswatch.ml