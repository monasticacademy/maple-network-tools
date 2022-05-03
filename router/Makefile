
# pull configuration from router to this repository
pull:
	ssh microtik /export file=router.rsc
	scp microtik:router.rsc .

# Compare the live configuration of the router to router.rsc
diff:
	ssh microtik /export file=live.rsc
	scp microtik:live.rsc /tmp/live.rsc
	diff router.rsc /tmp/live.rsc


# does not really work yet
push:
	echo "not implemented"
#	scp router.rsc microtik:
#	ssh microtik /import file=router.rsc
