import { describe, it, expect } from "vitest";
import {
  NAV_COMMANDS,
  ACTION_COMMANDS,
  filterCommands,
} from "./commands";

describe("NAV_COMMANDS", () => {
  it("has unique IDs", () => {
    const ids = NAV_COMMANDS.map((c) => c.id);
    expect(new Set(ids).size).toBe(ids.length);
  });

  it("all have navigation category", () => {
    for (const cmd of NAV_COMMANDS) {
      expect(cmd.category).toBe("navigation");
    }
  });

  it("covers expected routes", () => {
    const ids = NAV_COMMANDS.map((c) => c.id);
    expect(ids).toContain("nav:dashboard");
    expect(ids).toContain("nav:queue");
    expect(ids).toContain("nav:system");
    expect(ids).toContain("nav:indexers");
  });
});

describe("ACTION_COMMANDS", () => {
  it("has unique IDs", () => {
    const ids = ACTION_COMMANDS.map((c) => c.id);
    expect(new Set(ids).size).toBe(ids.length);
  });

  it("all have a taskName", () => {
    for (const cmd of ACTION_COMMANDS) {
      expect(cmd.taskName).toBeTruthy();
    }
  });
});

describe("filterCommands", () => {
  it("returns all commands when query is empty", () => {
    expect(filterCommands(NAV_COMMANDS, "")).toEqual(NAV_COMMANDS);
  });

  it("matches by label (case-insensitive)", () => {
    const result = filterCommands(NAV_COMMANDS, "dash");
    expect(result.length).toBe(1);
    expect(result[0].id).toBe("nav:dashboard");
  });

  it("matches by keyword", () => {
    const result = filterCommands(NAV_COMMANDS, "torznab");
    expect(result.length).toBe(1);
    expect(result[0].id).toBe("nav:indexers");
  });

  it("returns empty array when nothing matches", () => {
    expect(filterCommands(NAV_COMMANDS, "zzzznothing")).toEqual([]);
  });

  it("matches partial keyword", () => {
    const result = filterCommands(ACTION_COMMANDS, "rss");
    expect(result.length).toBe(1);
    expect(result[0].id).toBe("action:rss-sync");
  });

  it("is case-insensitive for keywords", () => {
    const result = filterCommands(NAV_COMMANDS, "QUEUE");
    expect(result.length).toBe(1);
    expect(result[0].id).toBe("nav:queue");
  });
});
