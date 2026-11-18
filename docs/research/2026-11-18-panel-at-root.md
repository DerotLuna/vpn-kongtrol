<img src="../../assets/brand/logo-kong-64.png" alt="VPN Kongtrol" width="48" align="left" />

# Panel at the site root: redirects, fragments and CSP

Version 1.0.0 · 2026-11-18

Backs [ADR-0043](../adr/0043-new-dashboard-replaces-classic.md). Sources accessed 2026-11-18.

## Questions

1. Does a redirect from `/app/x#k=<nonce>` to `/x` keep the one-time sign-in fragment?
2. Which status code for a moved page address?
3. Can the panel drop `'unsafe-inline'` from `script-src` and `style-src`?
4. How to serve one inline style on the "UI not built" page without `'unsafe-inline'`?

## Findings

1. **Fragments survive redirects.** RFC 9110 §10.2.2 (Location): if the `Location` value has no
   fragment, the user agent treats it as if it "inherits the fragment component of the URI
   reference used to generate the target URI". The fragment is never sent to the server, so the
   daemon cannot add it; it only has to leave it out of `Location`.
   https://www.rfc-editor.org/rfc/rfc9110.html#section-10.2.2 — verified by a Playwright run
   against a local daemon: `/app/rules#k=<nonce>` ended on `/rules`, signed in.
2. **308 Permanent Redirect** (RFC 9110 §15.4.9) says the move is permanent and, unlike 301,
   keeps the method. Pages are only GET/HEAD, so 301 would also work; 308 is the exact meaning.
   https://www.rfc-editor.org/rfc/rfc9110.html#section-15.4.9
3. **Inline code.** The Vite build of `web/ui` emits no inline `<script>` or `<style>` (checked in
   `web/ui/dist/index.html`: one `<script type="module" src>`, stylesheets by `<link>`). React sets
   `style` props on the element's `style` object; MDN (`style-src`): "styles properties that are
   set directly on the element's `style` property will not be blocked"; only `setAttribute('style')`
   and `style.cssText` are. https://developer.mozilla.org/en-US/docs/Web/HTTP/Reference/Headers/Content-Security-Policy/style-src
   (React's exact DOM calls: **no verificado** in React's source; the Playwright run logged no
   CSP violation on any page.) The tray flyout's injected script uses WebView2
   `AddScriptToExecuteOnDocumentCreated`, which runs outside the page's CSP (**no verificado**
   in Microsoft's docs; the flyout smoke test serves its own page without this CSP).
4. **Hash-source.** CSP Level 3 allows `'sha256-<base64>'` in `style-src` for an inline block
   whose content hashes to that value. https://www.w3.org/TR/CSP3/#grammardef-hash-source
5. **Open redirect.** Browsers treat a path starting with `/\` like `//host` (WHATWG URL parser
   treats `\` as `/` in special schemes). https://url.spec.whatwg.org/#special-authority-slashes-state
   The redirect sends any target with a backslash to `/`.

## Precedents

- SPA fallback (unknown non-file path → `index.html`, missing file → 404) is the pattern this
  repo already used under `/app/` ([Go-embedded SPA](2026-11-17-go-embedded-spa.md)); serving it
  at the root changes only the prefix.
