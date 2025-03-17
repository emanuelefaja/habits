---
title: "The Habits Company"
slug: "the-habits-company"
date: 2025-03-17T09:00:00Z
emoji: "📈"
description: "Looking at habits from a first principles perspective."
tags: ["building"]
---

Okay, so three months since I said [Hello World](/blog/hello) and introduced **habits**, a free, open-source habit tracker. 

The journey has been mostly calm, with coding in the evenings or weekends. I have learned a lot of Golang, had to refactor a bunch of things, but it's mostly gone pretty well!  

However, I realized that I want to go beyond a habit tracker, and create **The Habits Company**, reflecting a broader mission to help people build better habits through not just tracking, but education, resources, and community.

With 150 users and over 4,000 habit logs, I have gained some interesting insights into habit formation:

- **Less is more:** Users who start with 4-8 habits consistently stick around, while those who create 15-20 habits right away have the highest dropout rates. This aligns perfectly with research showing that habit overload leads to abandonment.

- **Variety matters:** I've expanded from simple yes/no habits to include numeric tracking, choice-based habits, and even set/rep tracking for exercise. This flexibility allows for more meaningful data and personalized insights.

- **Visualization drives motivation:** The individual habit pages now show a lot of data such as streaks, best days, and both cumulative and non-cumulative graphs. Personally, this gives me a bit of motivation to do my habits and see the numbers go the right way.

I experienced quite deeply with with the goals module. 

When I introduced it, setting measurable targets for my habits made a noticeable difference in my consistency. 

However, after the first month, I encountered some bugs that prevented proper tracking due to some timezone and date handling issues, and I was not able to figure this out for close to a month. 

Without the daily progress visibility, I found myself missing most of my targets that month. After fixing these issues in March, I immediately noticed how much easier it became to stay on track again. 

This personal experience reinforced something I already suspected: what gets measured gets managed, and proper visualization of progress is crucial for sustained motivation.

Building in public comes with its share of surprises. 

I faced hundreds of spam registration attacks, which forced me to temporarily close registrations, and then implement rate limiting and even a small math test during registration. This has pretty much stopped the attacks, and was a good lesson in proactive security. I'll be thinking more about form security in future updates to ensure this type of thing doesn't happen again.

One of the most rewarding moments came when a user who had forgotten his password reached out to me. At that time, I hadn't yet implemented a password reset function. Instead of just asking me to fix his account, he took matters into his own hands – contributing code to implement an admin password reset feature! 

This became our first external contribution to the codebase, and I was able to use his new feature to reset his password. It was a perfect demonstration of why open source matters – users becoming contributors, solving their own problems while improving the platform for everyone.

## From Tracking to Transformation

The most significant insight from the first three months? Tracking alone isn't enough.

Most people start habit trackers with enthusiasm but gradually disengage. The missing piece isn't better tracking – it's better guidance, education, and support throughout the habit-building journey. Just like [habits themselves are not enough](/blog/whatarehabits), it's about changing your identity, who you really are. 

That's why I think it's worth expanding beyond just a habit tracker. I'll do this following the typical "value ladder" where there are various products from free to paid.

- Various free email courses
- Structured courses on habit formation, both self paced and cohorts.
- Evidence-based resources and videos on making habits stick
- A more comprehensive approach to behavior change

## What's Next

I'll continue improving the core tracker – now at version v0.3.1 after dozens of incremental updates – while developing these new offerings. The tracker will always remain free and open-source, true to my original vision.

On the technical side, I'm genuinely excited about the new challenges ahead – building robust email campaigns, creating an interactive course platform, and perhaps even developing a community forum. Each of these presents unique engineering problems to solve, and as someone who loves to learn, that's half the fun of this journey! Golang is...quite beautiful and very logical. 

A key principle for The Habits Company is that everything will remain open source and available on GitHub – even the paid courses. This ensures that if people cannot afford to pay, they can always download the source code and run it locally. I'll also provide scholarships for anyone who genuinely cannot afford the paid options, and do not have the technical skills to run our platform locally, but would benefit from them.

My goal with The Habits Company is to bridge the gap between reading about habits and actually implementing lasting change. I have seen that the right combination of tracking, education, and community support creates a powerful foundation for personal growth.

And as always, the complete source code is available for anyone who wants to contribute or learn.

Here's to your better habits!

Manny


