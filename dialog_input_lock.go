package main

const dialogInputLockTicks = 12

func lockDialogInput(ticks *int) {
	*ticks = dialogInputLockTicks
}

func consumeDialogInputLock(ticks *int) bool {
	if *ticks > 0 {
		*ticks--
		return true
	}
	return false
}
