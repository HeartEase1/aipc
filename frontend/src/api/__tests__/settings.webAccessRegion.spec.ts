import { beforeEach, describe, expect, it, vi } from "vitest";

const { get, put } = vi.hoisted(() => ({
  get: vi.fn(),
  put: vi.fn(),
}));

vi.mock("@/api/client", () => ({
  apiClient: { get, put },
}));

import {
  getWebAccessRegionSettings,
  updateWebAccessRegionSettings,
} from "@/api/admin/settings";

describe("admin WebUI region access settings API", () => {
  beforeEach(() => {
    get.mockReset();
    put.mockReset();
  });

  it("uses the dedicated region access endpoints", async () => {
    const settings = { block_mainland_china: true };
    get.mockResolvedValueOnce({ data: settings });
    put.mockResolvedValueOnce({ data: settings });

    await expect(getWebAccessRegionSettings()).resolves.toEqual(settings);
    await expect(updateWebAccessRegionSettings(settings)).resolves.toEqual(settings);

    expect(get).toHaveBeenCalledWith("/admin/settings/web-access-region");
    expect(put).toHaveBeenCalledWith(
      "/admin/settings/web-access-region",
      settings,
    );
  });
});
