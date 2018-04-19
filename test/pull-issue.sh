#!/usr/bin/env bash
wget --recursive --no-host-directories --cut-dirs 6 --reject index.html* --reject *.jp2 \
     --include-directories /data/batches/batch_uuml_thys_ver01/data/sn83045396/print/1912031701/ \
     https://chroniclingamerica.loc.gov/data/batches/batch_uuml_thys_ver01/data/sn83045396/print/1912031701/

mv 1912031701 sources/sftp/sn83045396-1912031701
