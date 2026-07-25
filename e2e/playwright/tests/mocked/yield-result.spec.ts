/**
 * E2E: вкладка «Результат» показывает факт доходности с бекенда.
 */

import { test, expect } from "@playwright/test";
import {
  gotoPortfolio,
  makeAdvice,
  makeEmptyPlan,
  makePerformance,
  makeTradingPortfolio,
  mockTradingPortfolioRoutes,
} from "./fixtures";

const PORTFOLIO_ID = "yield-result-1";

test.describe("Доходность — Результат", () => {
  test("показывает чистую прибыль, годовую % и полную стоимость", async ({ page }) => {
    const portfolio = makeTradingPortfolio(PORTFOLIO_ID, { name: "Yield Result" });
    await mockTradingPortfolioRoutes(page, PORTFOLIO_ID, portfolio, {
      plan: makeEmptyPlan({
        expected_xirr_pct: 18,
        total_net_profit_with_held_rub: 40_000,
        final_portfolio_value: 280_000,
      }),
      advice: makeAdvice({
        money_rub: 8_000,
        available_money_rub: 8_000,
        performance: makePerformance({
          total_value_rub: 103_000,
          net_profit_rub: 3_000,
          annual_yield_pct: 12.17,
          xirr_pct: 12.17,
        }),
      }),
    });

    await gotoPortfolio(page, PORTFOLIO_ID);

    await expect(page.getByTestId("yield-metrics")).toBeVisible({ timeout: 15_000 });
    await expect(page.getByRole("tab", { name: "Результат" })).toBeVisible();

    await page.getByRole("tab", { name: "Результат" }).click();

    const panel = page.getByTestId("performance-metrics");
    await expect(panel).toBeVisible();
    await expect(panel.getByText("Чистая прибыль")).toBeVisible();
    await expect(page.getByTestId("performance-net-profit")).toContainText("3");
    await expect(page.getByTestId("performance-net-profit")).toContainText("000");
    await expect(page.getByTestId("performance-annual-yield")).toHaveText("12.17%");
    await expect(page.getByTestId("performance-total-value")).toContainText("103");
    await expect(page.getByTestId("performance-total-value")).toContainText("000");
  });
});
