// Тесты <MemberListItem />.

import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { MemberListItem } from "@/features/rooms/components/MemberListItem";
import { ru } from "@/shared/lib/i18n/ru";

import type { Member } from "@/features/rooms/types";

const otherMember: Member = {
  userId: "abcdef12-3456-7890-1234-567890abcdef",
  role: "member",
  joinedAt: "2025-01-01T00:00:00Z",
};

const ownerMember: Member = {
  userId: "11111111-2222-3333-4444-555555555555",
  role: "owner",
  joinedAt: "2025-01-01T00:00:00Z",
};

const adminMember: Member = {
  userId: "99999999-8888-7777-6666-555555555555",
  role: "admin",
  joinedAt: "2025-01-01T00:00:00Z",
};

function renderItem(node: JSX.Element): void {
  render(<ul>{node}</ul>);
}

describe("<MemberListItem />", () => {
  it("чужой member — отображает UUID-placeholder (первые 6 символов uppercase)", () => {
    renderItem(<MemberListItem member={otherMember} isMe={false} />);
    // "abcdef" без дефисов uppercase = "ABCDEF" (slice 6).
    expect(screen.getByText("ABCDEF")).toBeInTheDocument();
  });

  it("isMe=true с meUsername → показывает username вместо placeholder", () => {
    renderItem(
      <MemberListItem member={otherMember} isMe={true} meUsername="john" />,
    );
    expect(screen.getByText("john")).toBeInTheDocument();
    expect(screen.queryByText("ABCDEF")).not.toBeInTheDocument();
  });

  it("owner — badge владелец с aria-label", () => {
    renderItem(<MemberListItem member={ownerMember} isMe={false} />);
    expect(screen.getByText(ru.rooms.roleOwner)).toBeInTheDocument();
    expect(
      screen.getByLabelText(`роль: ${ru.rooms.roleOwner}`),
    ).toBeInTheDocument();
  });

  it("admin — badge админ", () => {
    renderItem(<MemberListItem member={adminMember} isMe={false} />);
    expect(screen.getByText(ru.rooms.roleAdmin)).toBeInTheDocument();
  });

  it("member — без badge", () => {
    renderItem(<MemberListItem member={otherMember} isMe={false} />);
    expect(screen.queryByText(ru.rooms.roleOwner)).not.toBeInTheDocument();
    expect(screen.queryByText(ru.rooms.roleAdmin)).not.toBeInTheDocument();
  });
});
