to make ts work you need to:

1. Install sleepwatcher
` brew install sleepwatcher
`
2. Create a .wakeup file in your home directory
`echo "~/random-workflows-that-actually-are-bots/bin/wake-sleep wake" > ~/.wakeup
`
3. Create a .sleep file in your home directory
`echo "~/random-workflows-that-actually-are-bots/bin/wake-sleep sleep
" > ~/.sleep
`
4. Activate the service
`brew services start sleepwatcher
`

uh oops you need to have the SLACK_WORKFLOW_BOT_TOKEN env var set too :)
