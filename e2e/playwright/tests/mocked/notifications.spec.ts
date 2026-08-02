/**
 * E2E: portfolio notifications panel, signals tab, and global bell inbox.
 */

import { test, expect } from "@playwright/test";
import {
  gotoPortfolio,
  makeTradingPortfolio,
  mockConfig,
  mockTradingPortfolioRoutes,
  seedAuth,
} from "./fixtures";

const PORTFOLIO_ID = "trading-portfolio-notify";

type Notif = {
  id: string;
  fingerprint: string;
  portfolio_id: string;
  portfolio_name?: string;
  kind: string;
  payload: Record<string, unknown>;
  urgency: string;
  created_at: string;
  read_at: string | null;
  dismissed_at: string | null;
  is_unread: boolean;
};

const baseNotifications: Notif[] = [
  {
    id: "notif-1",
    fingerprint: "fp-1",
    portfolio_id: PORTFOLIO_ID,
    portfolio_name: "Notifications E2E",
    kind: "put_offer_action",
    payload: {
      isin: "RU000PO",
      name: "Put Offer Bond",
      reason: "Окно подачи по пут-оферте открыто — подайте заявку на досрочное погашение.",
    },
    urgency: "soon",
    created_at: "2026-07-28T10:00:00+00:00",
    read_at: null,
    dismissed_at: null,
    is_unread: true,
  },
  {
    id: "notif-2",
    fingerprint: "fp-2",
    portfolio_id: PORTFOLIO_ID,
    portfolio_name: "Notifications E2E",
    kind: "sector_concentration",
    payload: {
      isin: "sectors:overweight",
      name: "Концентрация секторов",
      reason: "Сектор «financial» занимает 42.0% портфеля (лимит 35%). Сигнал модели: концентрация выше лимита стратегии.",
    },
    urgency: "normal",
    created_at: "2026-07-28T10:01:00+00:00",
    read_at: null,
    dismissed_at: null,
    is_unread: true,
  },
  {
    id: "notif-3",
    fingerprint: "fp-3",
    portfolio_id: PORTFOLIO_ID,
    portfolio_name: "Notifications E2E",
    kind: "spread_anomaly",
    payload: {
      isin: "RU000SPREAD1",
      name: "Spread Bond",
      reason:
        "Кредитный спред расширился относительно похожих бумаг: 15.0 п.п. vs медиана 5.0 п.п. (Δ 10.0 п.п., peers 6).",
    },
    urgency: "normal",
    created_at: "2026-07-28T10:02:00+00:00",
    read_at: null,
    dismissed_at: null,
    is_unread: true,
  },
  {
    id: "notif-4",
    fingerprint: "fp-4",
    portfolio_id: PORTFOLIO_ID,
    portfolio_name: "Notifications E2E",
    kind: "sector_stress",
    payload: {
      isin: "RU000SECTOR1",
      name: "Sector Stress Bond",
      reason: "Похоже на секторное давление: бумага падает вместе с похожими бумагами из сектора.",
      bond_change_7d_pct: -4.2,
      sector_change_7d_pct: -3.8,
    },
    urgency: "normal",
    created_at: "2026-07-28T10:03:00+00:00",
    read_at: null,
    dismissed_at: null,
    is_unread: true,
  },
  {
    id: "notif-5",
    fingerprint: "fp-5",
    portfolio_id: PORTFOLIO_ID,
    portfolio_name: "Notifications E2E",
    kind: "turbo_entry",
    payload: {
      isin: "RU000TURBO1",
      name: "Turbo Bond",
      reason: "Turbo-entry: сектор в панике, а бумага просела сильнее сектора без ухудшения рейтинга.",
      suggested_price_pct: 99.1,
      lots: 1,
    },
    urgency: "normal",
    created_at: "2026-07-28T10:04:00+00:00",
    read_at: null,
    dismissed_at: null,
    is_unread: true,
  },
];

async function mockNotificationRoutes(page: import("@playwright/test").Page, store: Notif[]) {
  await page.route(`**/api/v1/portfolios/${PORTFOLIO_ID}/notifications**`, async (route) => {
    const url = new URL(route.request().url());
    const unreadOnly = url.searchParams.get("unread_only") === "true";
    const list = unreadOnly
      ? store.filter((n) => n.is_unread && n.dismissed_at == null)
      : store.filter((n) => n.dismissed_at == null);
    await route.fulfill({ json: { notifications: list } });
  });

  await page.route("**/api/v1/notifications", async (route) => {
    if (route.request().method() !== "GET") {
      await route.fallback();
      return;
    }
    const list = store.filter((n) => n.dismissed_at == null);
    await route.fulfill({
      json: {
        notifications: list,
        unread_count: list.filter((n) => n.is_unread).length,
      },
    });
  });

  await page.route("**/api/v1/notifications/*/read", async (route) => {
    const id = route.request().url().split("/").slice(-2)[0];
    const idx = store.findIndex((n) => n.id === id);
    if (idx >= 0) {
      store[idx] = {
        ...store[idx],
        is_unread: false,
        read_at: "2026-07-28T12:00:00+00:00",
      };
    }
    await route.fulfill({ status: 204, body: "" });
  });
}

test.describe("Уведомления портфеля", () => {
  test("разделяет операционные уведомления и вкладку сигналов", async ({ page }) => {
    const store = structuredClone(baseNotifications);
    const portfolio = makeTradingPortfolio(PORTFOLIO_ID, {
      name: "Notifications E2E",
    });
    await mockTradingPortfolioRoutes(page, PORTFOLIO_ID, portfolio);
    await mockNotificationRoutes(page, store);

    await gotoPortfolio(page, PORTFOLIO_ID);
    const panel = page.getByTestId("notifications-panel");
    await expect(panel).toBeVisible();
    await expect(panel.getByTestId("notifications-unread-badge")).toHaveText("2");
    await expect(panel.getByText("Put Offer Bond")).toBeVisible();
    await expect(panel.getByTestId("notification-notif-1-created-at")).toContainText("28 июля");
    await expect(panel.getByText("Окно подачи по пут-оферте открыто")).toBeVisible();
    await expect(panel.getByText("Концентрация секторов")).toBeVisible();
    await expect(panel.getByText("Финансы")).toBeVisible();
    await expect(panel.getByText("Spread Bond")).not.toBeVisible();
    await expect(panel.getByText("Turbo Bond")).not.toBeVisible();

    await page.getByRole("tab", { name: /Сигналы/ }).click();
    const signalsPanel = page.getByTestId("signals-panel");
    await expect(signalsPanel).toBeVisible();
    await expect(signalsPanel.getByText("Сигналы рынка")).toBeVisible();
    await expect(signalsPanel.getByText("Spread Bond")).toBeVisible();
    await expect(signalsPanel.getByText("Sector Stress Bond")).toBeVisible();
    await expect(signalsPanel.getByText("Turbo Bond")).toBeVisible();
    await expect(signalsPanel.getByText("Put Offer Bond")).not.toBeVisible();
  });

  test("кнопка Прочитано убирает уведомление из панели портфеля", async ({ page }) => {
    const store = structuredClone(baseNotifications);
    const portfolio = makeTradingPortfolio(PORTFOLIO_ID, {
      name: "Notifications E2E",
    });
    await mockTradingPortfolioRoutes(page, PORTFOLIO_ID, portfolio);
    await mockNotificationRoutes(page, store);

    await gotoPortfolio(page, PORTFOLIO_ID);
    const panel = page.getByTestId("notifications-panel");
    await expect(panel.getByTestId("notification-notif-1")).toBeVisible();

    await panel
      .getByTestId("notification-notif-1")
      .getByRole("button", { name: "Прочитано" })
      .click();

    await expect(panel.getByTestId("notification-notif-1")).toHaveCount(0);
    await expect(panel.getByText("Put Offer Bond")).toHaveCount(0);
    await expect(panel.getByTestId("notifications-unread-badge")).toHaveText("1");
  });
});

test.describe("Колокольчик уведомлений", () => {
  async function openNotificationsInbox(page: import("@playwright/test").Page) {
    const store = structuredClone(baseNotifications);
    store[0] = {
      ...store[0],
      is_unread: false,
      read_at: "2026-07-28T11:00:00+00:00",
    };

    await seedAuth(page);
    await mockConfig(page);
    await mockNotificationRoutes(page, store);
    await page.route("**/api/v1/portfolios/", async (route) => {
      await route.fulfill({
        json: [makeTradingPortfolio(PORTFOLIO_ID, { name: "Notifications E2E" })],
      });
    });
    await page.route("**/api/v1/favorites/**", async (route) => {
      await route.fulfill({ json: { isins: [], count: 0 } });
    });
    await page.route("**/api/v1/bonds/**", async (route) => {
      await route.fulfill({
        json: {
          bonds: [],
          source: "mock",
          count: 0,
          total: 0,
          page: 1,
          page_size: 50,
        },
      });
    });

    await page.goto("/");
    const bell = page.locator('[data-testid="notifications-bell"]:visible');
    await expect(bell).toBeVisible();
    await expect(bell).toBeEnabled();
    await expect(bell.getByTestId("notifications-bell-badge")).toHaveText("4");

    // Inbox must be closed before click.
    await expect(page.getByTestId("notifications-inbox-list")).toHaveCount(0);

    await bell.click();

    const inbox = page
      .getByTestId("notifications-inbox-popover")
      .or(page.getByTestId("notifications-inbox-sheet"));
    await expect(inbox).toBeVisible({ timeout: 5_000 });
    await expect(inbox.getByTestId("notifications-inbox-list")).toBeVisible();
    return inbox;
  }

  test("клик по колокольчику открывает inbox", async ({ page }) => {
    const inbox = await openNotificationsInbox(page);
    await expect(inbox.getByText("Уведомления").first()).toBeVisible();
    await expect(inbox.getByTestId("inbox-notification-notif-1")).toBeVisible();
    await expect(inbox.getByTestId("inbox-notification-notif-3")).toBeVisible();
  });

  test("inbox показывает все уведомления, источник и кнопку Прочитано", async ({ page }) => {
    const inbox = await openNotificationsInbox(page);
    await expect(inbox.getByText("Радар · Notifications E2E").first()).toBeVisible();
    await expect(inbox.getByText("Notifications E2E").first()).toBeVisible();
    await expect(
      inbox.getByTestId("inbox-notification-notif-1").getByRole("button", { name: "Прочитано" }),
    ).toHaveCount(0);
    await expect(
      inbox.getByTestId("inbox-notification-notif-2").getByRole("button", { name: "Прочитано" }),
    ).toBeVisible();
  });

  test("клик по уведомлению ведёт в портфель", async ({ page }) => {
    const inbox = await openNotificationsInbox(page);
    await mockTradingPortfolioRoutes(
      page,
      PORTFOLIO_ID,
      makeTradingPortfolio(PORTFOLIO_ID, { name: "Notifications E2E" }),
    );
    await page.route(`**/api/v1/portfolios/${PORTFOLIO_ID}/notifications**`, async (route) => {
      await route.fulfill({ json: { notifications: [] } });
    });

    await inbox.getByTestId("inbox-notification-notif-3").click();
    await expect(page).toHaveURL(new RegExp(`/portfolio/${PORTFOLIO_ID}`));
    await expect(page.getByTestId("notifications-inbox-list")).toHaveCount(0);
  });
});
