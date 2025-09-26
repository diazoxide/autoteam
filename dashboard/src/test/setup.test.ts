import { describe, it, expect } from "vitest";

describe("Test Setup", () => {
  it("should run basic tests", () => {
    expect(1 + 1).toBe(2);
  });

  it("should have access to testing environment", () => {
    expect(typeof window).toBe("object");
    expect(typeof document).toBe("object");
  });
});
