// Kongtrol UI strings — shape of one string module · v1.0.0 · 2026-11-18

/** EN (and any future language) must have exactly the ES keys of its module. */
export type Strings<ES> = Record<keyof ES, string>
