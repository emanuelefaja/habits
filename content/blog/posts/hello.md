---
title: "Hello, World!"
slug: "hello"
date: 2024-12-16T09:00:00Z
emoji: "👋"
description: "Why I started a free, open source, minimalist habit tracker."
tags: ["building"]
author: "Manny"
---


So, here we go! 

First, a little about me. 

My name is Manny. 

I'm the founder of [Blue](https://blue.cc), a project and process management platform that's used by over 14,000 companies across the world. I'm also Senior Product Manager at the Executive Office of [UNDP (United Nations Development Programme)](https://undp.org).  

## Why I started a habit tracker

This idea of mine is actually a few years old, and I even coded a command-line version of this a few years ago!

![](/blog/media/habits-terminal.webp)

I started to (re)track my habits a few months ago using plain old pen and paper, creating a monthly grid with the days running across the top, and my habits on the left as rows.

The simple act of actually tracking what I wanted to achieve and turn into habits helped. However, I wanted to have a more visual way to track the data behind my habits.

The main issue is that most habit trackers allow you to simply track whether you did something or not, but I wanted to be able to track the data behind my habits, such as how many sets and reps I did, or how much weight I lifted, how many km I ran, etc.  I also wanted a really nice looking monthly calendar/grid view of my habits.

I explored some of the existing options, but didn't find anything that I really liked. I have been learning [Golang](https://go.dev) for a while and I thought it was time that I tried to build something end to end. 

Much of this project was shaped during my walks around the walls of Lucca, in Tuscany. There's something special about walking the same historic path that people have traversed for centuries - a complete 4.2km circle atop Renaissance-era walls. These walks became my daily habit, and the consistency of this routine helped me think deeply about what makes habits stick.

The walls of Lucca have their own story about habits and consistency. Built in the 16th and 17th centuries, they've remained intact through centuries of change, standing as a testament to careful maintenance and preservation. 

In a way, they embody what I hope to achieve with this project - something sustainable, well-crafted, and built to last.

Building this has been a learning experience in many ways. 

Beyond the technical challenges, it's taught me about my own habit-forming process. I've realized that the tools we use shape our behavior in subtle ways, and I hope to create something that encourages sustainable, long-term habit formation rather than unsustainable perfectionism.


## How is this free?


Taking money out of the equation takes a lot of pressure off. There is no "metric of success" here, I just want to make something that is useful to my life, and hopefully it can be useful to other people too. 

The costs are pretty low as well. My main cost was $2,500 for the domain habits.co. This was mostly as a forcing function to get myself to work on this project as I can be quite lazy without some form of motivation. 

The hosting is pretty cheap ($7/month) and as this runs using Go, which is very efficient, I expect to be able to host thousands of users (nice problem to have!) for very little cost. 


## Why is this open source?

I've been a huge fan of open source for a long time, and I think it's a great way to build things. That said, I've never actually released anything open source before, so this is somewhat of a learning experience for me. I think sometimes habits can be quite personal, and I want to make sure that people can self-host or run this on their personal computer if they want to. 


## Current Features

Right now, the core features are intentionally minimal.

You can track four types of habits:

1. Yes/No — did you do it or not?
2. Number — how many times did you do it?
3. Set/Reps — how many sets and reps did you do?
4. Choice — Choose from a list of options

You can export your data to CSV at any time.

You can visualize your monthly progress across all your habits, and for each habit you can see your progress across the entire year plus some simple key statistics. 

There are no ads, no tracking, no analytics, no upsells, or notifications. This latter point is something I am thinking about, likely there will be some form of reminders in the future. 

## What's next?

There is a [roadmap](/roadmap) where you can see what I'm working on. Key things are going to be adding a few more habit types, adding more visualizations within the individual habit view, and also adding notes to habit logs, which appears to be a popular feature in other habit trackers. 

I believe in starting small and focusing on the essentials. Rather than building a feature-packed app that tries to do everything, I want to create something that does a few things really well.

And if you're a developer, the code is open source - contributions and feedback are always welcome.

Regards,

Manny
