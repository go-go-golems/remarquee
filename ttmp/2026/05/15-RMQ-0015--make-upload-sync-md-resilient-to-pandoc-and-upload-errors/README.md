# Make upload sync/md resilient to pandoc and upload errors

This is the document workspace for ticket RMQ-0015.

## Structure

- **design/**: Design documents and architecture notes
- **reference/**: Reference documentation and API contracts
- **playbooks/**: Operational playbooks and procedures
- **scripts/**: Utility scripts and automation
- **sources/**: External sources and imported documents
- **various/**: Scratch or meeting notes, working notes
- **archive/**: Optional space for deprecated or reference-only artifacts

## Getting Started

Use docmgr commands to manage this workspace:

- Add documents: `docmgr doc add --ticket RMQ-0015 --doc-type design-doc --title "My Design"`
- Import sources: `docmgr import file --ticket RMQ-0015 --file /path/to/doc.md`
- Update metadata: `docmgr meta update --ticket RMQ-0015 --field Status --value review`
