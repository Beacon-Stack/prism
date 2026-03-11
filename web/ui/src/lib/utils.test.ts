import { describe, it, expect } from "vitest";
import { formatBytes, formatDate } from "./utils";

describe("formatBytes", () => {
  it("formats 0 bytes", () => {
    expect(formatBytes(0)).toBe("0 B");
  });

  it("formats bytes", () => {
    expect(formatBytes(500)).toBe("500 B");
  });

  it("formats kilobytes", () => {
    expect(formatBytes(1024)).toBe("1 KB");
    expect(formatBytes(1536)).toBe("1.5 KB");
  });

  it("formats megabytes", () => {
    expect(formatBytes(1_048_576)).toBe("1 MB");
  });

  it("formats gigabytes", () => {
    expect(formatBytes(1_073_741_824)).toBe("1 GB");
    expect(formatBytes(8_589_934_592)).toBe("8 GB");
  });

  it("formats terabytes", () => {
    expect(formatBytes(1_099_511_627_776)).toBe("1 TB");
  });
});

describe("formatDate", () => {
  it("formats an ISO date string", () => {
    const result = formatDate("2024-06-15T14:30:00Z");
    expect(result).toBeTruthy();
    expect(result).toContain("Jun");
    expect(result).toContain("15");
  });

  it("includes year when includeYear is true", () => {
    const result = formatDate("2024-06-15T14:30:00Z", true);
    expect(result).toContain("2024");
  });

  it("omits year by default", () => {
    const result = formatDate("2024-06-15T14:30:00Z");
    expect(result).not.toContain("2024");
  });
});
