import { mount } from "@vue/test-utils";
import { describe, expect, it, vi } from "vitest";

import GroupBlockedModelsPanel from "../GroupBlockedModelsPanel.vue";

vi.mock("vue-i18n", () => ({
  useI18n: () => ({
    t: (key: string, params?: Record<string, unknown>) =>
      params ? `${key}:${JSON.stringify(params)}` : key,
  }),
}));

describe("GroupBlockedModelsPanel", () => {
  it("searches candidates and emits toggle and clear actions", async () => {
    const wrapper = mount(GroupBlockedModelsPanel, {
      props: {
        models: ["gpt-5.6-sol", "gpt-5.6-luna", "legacy-gpt"],
        blockedModels: ["gpt-5.6-luna"],
      },
    });

    const search = wrapper.get('input[type="search"]');
    await search.setValue("luna");
    expect(wrapper.text()).toContain("gpt-5.6-luna");
    expect(wrapper.text()).not.toContain("gpt-5.6-sol");

    await wrapper.get('input[type="checkbox"]').trigger("change");
    expect(wrapper.emitted("toggle-model")).toEqual([["gpt-5.6-luna"]]);

    const clearButton = wrapper.findAll("button").find(button =>
      button.text().includes("admin.groups.blockedModels.clear"),
    );
    expect(clearButton).toBeDefined();
    await clearButton!.trigger("click");
    expect(wrapper.emitted("clear")).toHaveLength(1);
  });
});
