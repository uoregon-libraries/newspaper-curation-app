---
title: Adding Titles
weight: 20
description: Back-end details for adding a new title
---

Adding a title to NCA is very simple in cases where the title already exists
somewhere external (e.g., Library of Congress), but can take a lot more work
for titles not indexed elsewhere. Here's our process:

- Provision a *real* record, including things like an LCCN
  - This is a must for us, otherwise our [Historic Oregon Newspapers](https://oregonnews.uoregon.edu/)
    site will misrepresent information that could be extremely confusing to
    end-users looking for more details.
  - Unfortunately this is black magic to me - we have a librarian who handles
    this and knows the right people to contact
- Generate MARC XML for the title(s)
  - [MarcEdit](https://marcedit.reeset.net) is a popular choice for this
- Ingest the XML into ONI:
  - Put the MARC XML files into a filesystem location ONI can read
  - Use the ONI administration command `load_titles` to ingest, e.g.:
    `./manage.py load_titles /path/to/marcxml/`
- Point NCA to your local ONI server instead of, or in addition to, Library of
  Congress. This can be done by modifyting the NCA settings `MARC_LOCATION_1`
  and/or `MARC_LOCATION_2`. e.g.:
  ```
  MARC_LOCATION_1="https://oregonnews.uoregon.edu/lccn/{{lccn}}/marc.xml"
  MARC_LOCATION_2="https://chroniclingamerica.loc.gov/lccn/{{lccn}}/marc.xml"
  ```
