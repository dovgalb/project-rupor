// Полная карта роутов (Phase 5).
// Источник истины: docs/3_5_frontend/08-routes.md.

import { Outlet, createBrowserRouter, Navigate } from "react-router-dom";

import { AppShell } from "@/app/AppShell";
import { RedirectIfAuthenticated } from "@/app/routes/RedirectIfAuthenticated";
import { RequireAuth } from "@/app/routes/RequireAuth";
import { RequireChannelInRoom } from "@/app/routes/RequireChannelInRoom";
import { RequireMembership } from "@/app/routes/RequireMembership";
import { ChatRoutePage } from "@/pages/ChatRoutePage";
import { LoginPage } from "@/pages/LoginPage";
import { NotFoundPage } from "@/pages/NotFoundPage";
import { RegisterPage } from "@/pages/RegisterPage";
import { RoomPage } from "@/pages/RoomPage";
import { RoomsIndexPage } from "@/pages/RoomsIndexPage";

export const router = createBrowserRouter([
  { path: "/", element: <Navigate to="/rooms" replace /> },
  {
    path: "/login",
    element: (
      <RedirectIfAuthenticated>
        <LoginPage />
      </RedirectIfAuthenticated>
    ),
  },
  {
    path: "/register",
    element: (
      <RedirectIfAuthenticated>
        <RegisterPage />
      </RedirectIfAuthenticated>
    ),
  },
  {
    path: "/rooms",
    element: (
      <RequireAuth>
        <AppShell />
      </RequireAuth>
    ),
    children: [
      { index: true, element: <RoomsIndexPage /> },
      {
        path: ":roomId",
        element: (
          <RequireMembership>
            <Outlet />
          </RequireMembership>
        ),
        children: [
          { index: true, element: <RoomPage /> },
          {
            path: "channels/:channelId",
            element: (
              <RequireChannelInRoom>
                <ChatRoutePage />
              </RequireChannelInRoom>
            ),
          },
        ],
      },
    ],
  },
  { path: "*", element: <NotFoundPage /> },
]);
