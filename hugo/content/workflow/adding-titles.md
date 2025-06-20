---
title: Adding Titles
weight: 20
description: Back-end details for adding a new title
---

Adding a title to NCA is very simple in cases where the title already exists
somewhere external (e.g., Library of Congress), but can take a lot more work
for titles not indexed elsewhere.

## Existing Records

For existing records, you just go in to NCA and create a title, and it will
validate the LCCN with your configured external server (`MARC_LOCATION_1` and
`MARC_LOCATION_2` in NCA's settings). Usually this is Library of Congress, but
any location can be used if it serves up a MARC record based on an LCCN.

If you have titles in your production ONI, but they don't exist elsewhere, you
can just use ONI directly, as it exposes the MARC XML. Point NCA to your local
ONI server instead of, or in addition to, Library of Congress. This can be done
by modifyting the NCA settings `MARC_LOCATION_1` and/or `MARC_LOCATION_2`.
e.g., our setup looks like this:

```
MARC_LOCATION_1="https://oregonnews.uoregon.edu/lccn/{{lccn}}/marc.xml"
MARC_LOCATION_2="https://chroniclingamerica.loc.gov/lccn/{{lccn}}/marc.xml"
```

## New Records

If you have a totally new record that isn't indexed in LoC *or* your ONI
instance, you'll need to create a record and get it into NCA as well as your
ONI staging and production systems.

The process will likely be similar to ours, but you may have to adapt it.
Here's how we do it:

- Provision a *real* record, including things like an LCCN (a unique identifier
  provisioned by the library of congress).
  - This is a must for us, otherwise our [Historic Oregon Newspapers](https://oregonnews.uoregon.edu/)
    site will misrepresent information that could be extremely confusing to
    end-users looking for more details.
  - Unfortunately this is black magic to me - we have a librarian who handles
    this and knows the right people to contact.
- Generate MARC XML for the title(s)
  - [MarcEdit](https://marcedit.reeset.net) is a popular choice for this
- Upload the XML into NCA (Lists -> Titles, "Upload a MARC record"). This
  creates records in staging and production ONI instances as well as a record
  "stub" in NCA.
- Make sure you are using your ONI instance as one of the MARC locations in
  NCA's settings (see above) so that titles in NCA can be validated once
  they're loaded into ONI.

When uploading MARC records into NCA, note that they are queued up in the ONI
Agent (our custom ONI command runner which automates what used to be
command-line-only tasks)
