# Rooted reads for OpenVPN references

Access date: 2026-11-26. Supports [ADR-0127](../adr/0127-rooted-reference-reads.md).

- [Go: Traversal-resistant file APIs](https://go.dev/blog/osroot): checking
  paths before opening them is vulnerable to directory replacement; `os.Root`
  keeps operations under an opened directory. It does not prevent mount
  traversal, and it allows symlinks that stay within the root.
- [`os.Root`](https://pkg.go.dev/os#Root): on Windows an open root prevents
  the directory from being renamed; on Unix it remains attached to the opened
  directory when it is renamed.
- [`openat(2)`](https://man7.org/linux/man-pages/man2/openat.2.html):
  directory-relative open and `O_NOFOLLOW` protect the final path component.

Verification: the Linux test replaces an opened directory with a symlink and
checks that the file is read from the original directory. Another test checks
that a final symlink is refused. `os.Root.OpenFile` alone followed a final
symlink even with `O_NOFOLLOW` in the Linux test, so the Unix implementation
uses `openat` against an open directory descriptor. Windows and Linux package
tests passed; native macOS behavior remains unverified.
