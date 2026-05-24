package server

import "golang.org/x/sys/unix"

func SpliceFileToSocket(fileFD int, clientFD int, size int64) (int64, error) {

	var pipeFDs [2]int
	err := unix.Pipe(pipeFDs[:])

	if err != nil {
		return 0, err
	}

	defer unix.Close(pipeFDs[0])
	defer unix.Close(pipeFDs[1])

	var totalSpliced int64 = 0

	spliceFlags := unix.SPLICE_F_MOVE | unix.SPLICE_F_NONBLOCK

	for totalSpliced < size {
		remaining := size - totalSpliced

		nBytesIntoPipe, err := unix.Splice(fileFD, nil, pipeFDs[1], nil, int(remaining), spliceFlags)
		if err != nil {
			if err == unix.EAGAIN || err == unix.EWOULDBLOCK {
				continue
			}
			return totalSpliced, err
		}
		if nBytesIntoPipe == 0 {
			break
		}

		var pipeBytesWritten int64 = 0
		for pipeBytesWritten < nBytesIntoPipe {
			nBytesOutToSocket, err := unix.Splice(
				pipeFDs[0],
				nil,
				clientFD,
				nil,
				int(nBytesIntoPipe-pipeBytesWritten),
				spliceFlags,
			)
			if err != nil {
				if err == unix.EAGAIN || err == unix.EWOULDBLOCK {
					continue
				}
				return totalSpliced, err
			}
			pipeBytesWritten += nBytesOutToSocket
		}
		totalSpliced += nBytesIntoPipe
	}

	return totalSpliced, nil
}
