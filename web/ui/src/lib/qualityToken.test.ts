import { describe, expect, it } from "vitest";
import { hasQualityToken, setQualityToken } from "./qualityToken";

describe("setQualityToken", () => {
  const withQ = "{Movie Title} ({Release Year}) {Quality Full}";
  const without = "{Movie Title} ({Release Year})";

  it("removes the token and the space before it", () => {
    expect(setQualityToken(withQ, false)).toBe(without);
  });
  it("removes a token in the middle", () => {
    expect(setQualityToken("{Movie Title} {Quality Full} ({Release Year})", false)).toBe("{Movie Title} ({Release Year})");
  });
  it("appends the token once", () => {
    expect(setQualityToken(without, true)).toBe(withQ);
    expect(setQualityToken(withQ, true)).toBe(withQ);
  });
  it("leaves an empty format empty", () => {
    expect(setQualityToken("", true)).toBe("");
  });
  it("detects the token", () => {
    expect(hasQualityToken(withQ)).toBe(true);
    expect(hasQualityToken(without)).toBe(false);
  });
});
