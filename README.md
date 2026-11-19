# Tiny Cosmos

A weekly check-in app for friends and partners. Everyone answers the same prompt, answers unlock when the whole group is in, and your shared tiny cosmos grows with every week.

## The idea

Friends drift apart when life gets busy or when they live far away from each other. Tiny Cosmos gives a group a small, structured ritual to stay close.

Once a week, everyone in the group gets the same prompt, for example:

- "What are you excited about this week?"
- "What's been weighing on you lately?"
- "What made you laugh this week?"

Everyone answers on their own time. Answers stay hidden until the last person responds, then they're revealed to the whole group at once. Nobody performs for an audience, and nobody reads ahead.

It works for friend groups, families, and couples, especially when people are physically apart.

## How it works

1. **Create a cosmos.** Start a group and invite friends or a partner.
2. **Get the weekly prompt.** Everyone receives the same question.
3. **Answer asynchronously.** Each person's planet moves into line as they answer.
4. **Align.** When the last person answers, the system aligns and all answers unlock together.
5. **Reply.** Respond to each other's answers to keep the conversation going.

## The cosmos

Each group has a living visual that grows as the group keeps showing up together.

- **The group is a little solar system.** Each member is a planet orbiting a shared, glowing center.
- **Alignment is the reveal.** Planets line up as people answer, and answers unlock when the last one slides into place.
- **Replies draw constellation lines.** Responding to a friend's answer connects your two bodies with a line of light. Over time, every group traces a shape that belongs only to them.
- **Each week becomes a star.** The group's history is a night sky you can browse, and tapping a star brings back that week's prompt and answers.

### Growth stages

| Stage | Timeframe | What it looks like |
| --- | --- | --- |
| Spark | Week 1 | A single point of light as the group forms |
| Moon | Weeks 2–4 | The first bodies settle into orbit |
| System | Months 1–3 | Everyone has a path, and alignment becomes a ritual |
| Constellation | Months 3–12 | Replies trace a shape unique to the group |
| Galaxy | A year and beyond | A whole sky of shared weeks |

## Design principles

- **Growth, never guilt.** The cosmos only ever grows. Missed weeks don't cause decay, and a member who skips drifts onto a wider, dimmer orbit until they return. A friend going through a hard time should never be the reason the group's cosmos "dies."
- **Reward connection, not streaks.** Growth reflects consistency over time and real engagement, like replies, rather than unbroken streaks.
- **Together or not at all.** Answers are revealed only when everyone is in, so the moment is shared.
- **Quiet weeks still count.** A missed week appears in the sky as a dim "quiet week" star, still part of the group's shape.

## Visual identity

A night-sky palette with a warm gold center and one color per member.

| Role | Color |
| --- | --- |
| Night background | `#0E1330` |
| Surface | `#161C44` |
| Orbit rings and borders | `#2A3264` |
| Gold accent | `#F2C46B` |
| Core glow | `#FFE7A8` |
| Secondary text | `#B9BEDB` |
| Member colors | `#F08A6C`, `#5CC8B8`, `#A99CF0`, `#6FA8F5` |

Typography: **Space Grotesk** for headings and **Inter** for body text.

Motion should be subtle (slow orbits, a gently pulsing center, a twinkling current-week star) and respect the system's reduced-motion setting.

## Tech stack

- [Flutter](https://flutter.dev) for iOS and Android
- Backend, authentication and notifications: _TBD_

## Getting started

Prerequisites: the [Flutter SDK](https://docs.flutter.dev/get-started/install) and an iOS simulator, Android emulator, or physical device.

```bash
git clone <repo-url>
cd tiny-cosmos
flutter pub get
flutter run
```

## Roadmap

- [ ] Groups and invites
- [ ] Weekly prompt delivery and reminders
- [ ] Asynchronous answers with group-wide reveal
- [ ] Replies to answers
- [ ] Animated solar-system home screen with alignment
- [ ] "Your sky" history view
- [ ] Growth stages
- [ ] Prompt library
- [ ] Couples mode (two bodies orbiting each other)

## Status

Early concept and design. Not yet ready for use.