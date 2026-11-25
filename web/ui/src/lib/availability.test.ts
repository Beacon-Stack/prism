import { describe, expect, it } from "vitest";
import { availabilityLabel, availabilityOf } from "./availability";

describe("availabilityOf", () => {
  it("uses what the server says", () => {
    expect(availabilityOf({ availability: "available", status: "downloaded", monitored: true })).toBe("available");
  });

  it("falls back to the stored status for a server that doesn't send it", () => {
    expect(availabilityOf({ status: "downloaded", monitored: true })).toBe("downloaded");
    expect(availabilityOf({ status: "missing", monitored: true })).toBe("missing");
    expect(availabilityOf({ status: "announced", monitored: true })).toBe("upcoming");
    expect(availabilityOf({ status: "released", monitored: true })).toBe("wanted");
    expect(availabilityOf({ status: "released", monitored: false })).toBe("unmonitored");
  });

  it("ignores a state it doesn't know", () => {
    expect(availabilityOf({ availability: "teleported", status: "released", monitored: true })).toBe("wanted");
  });
});

describe("availabilityLabel", () => {
  it("shows the percentage while downloading", () => {
    expect(availabilityLabel("downloading", 0.256)).toBe("Downloading 25%");
    expect(availabilityLabel("downloading", 0)).toBe("Downloading");
    expect(availabilityLabel("available")).toBe("Available");
  });
});
