import { describe, expect, it } from "vitest";
import { grabbedAgo } from "./grabbedAgo";

const now = Date.parse("2026-11-25T18:00:00Z");
const ago = (hours: number) => new Date(now - hours * 3_600_000).toISOString();

describe("grabbedAgo", () => {
  it("says today for anything under a day, not 'Today ago'", () => {
    expect(grabbedAgo(ago(0), now)).toBe("today");
    expect(grabbedAgo(ago(5), now)).toBe("today");
    expect(grabbedAgo(ago(23), now)).toBe("today");
  });

  it("counts days, months and years", () => {
    expect(grabbedAgo(ago(24), now)).toBe("1d ago");
    expect(grabbedAgo(ago(24 * 12), now)).toBe("12d ago");
    expect(grabbedAgo(ago(24 * 75), now)).toBe("2mo ago");
    expect(grabbedAgo(ago(24 * 800), now)).toBe("2y ago");
  });

  it("copes with a clock that is slightly behind or a bad date", () => {
    expect(grabbedAgo(ago(-1), now)).toBe("today");
    expect(grabbedAgo("not a date", now)).toBe("today");
  });
});
