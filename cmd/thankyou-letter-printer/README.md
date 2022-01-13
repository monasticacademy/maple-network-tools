There is a Zapier automation that triggers every time a record is added to the donations table in Airtable that:
  - creates a Google Doc form letter
  - submits the ID of that Google Doc to the cloud function found in ../../cloudfunctions/thankyou-letter-helper
  - sends a Slack message to #donation-tracking

Next the cloud function pushes the document ID to a pubsub topic. The code in this directory listens for messages on that pubsub topic and:
 - downloads the Google Doc as a PDF
 - converts it to postscript
 - sends it to the letterhead printer in Manjushri

This is how form letters are automatically printed.
