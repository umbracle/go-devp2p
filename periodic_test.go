package devp2p

import (
	"testing"
	"time"
)

var (
	span = time.Duration(100 * time.Millisecond).Nanoseconds()
)

func testPeriodicDispatcher() *Dispatcher {
	d := NewDispatcher()
	d.SetEnabled(true)
	return d
}

type dummyJob struct {
	id string
}

func (d *dummyJob) ID() string {
	return d.id
}

func job(id string) Job {
	return &dummyJob{id}
}

func waitForEvent(d *Dispatcher, id string, t *testing.T) time.Duration {
	now := time.Now()
	evnt := <-d.Events()
	if evnt.ID() != id {
		t.Fatalf("expected %s but found %s", id, evnt.ID())
	}
	return time.Now().Sub(now)
}

func validPeriod(i, j time.Duration, t *testing.T) {
	d := i.Nanoseconds() - j.Nanoseconds()
	if d < 0 {
		d = d * -1
	}
	if d > span {
		t.Fatal("bad")
	}
}

func TestPeriodicDispatcherAddJob(t *testing.T) {
	d := testPeriodicDispatcher()

	if err := d.Add(job("a"), 1*time.Second); err != nil {
		t.Fatal(err)
	}
	if len(d.Tracked()) != 1 {
		t.Fatal("it should have 1 tracked jobs")
	}
}

func TestPeriodicDispatcherRemoveJob(t *testing.T) {
	d := testPeriodicDispatcher()

	if err := d.Add(job("a"), 1*time.Second); err != nil {
		t.Fatal(err)
	}
	if err := d.Remove("a"); err != nil {
		t.Fatal(err)
	}
	if len(d.Tracked()) != 0 {
		t.Fatal("it should have 0 tracked jobs")
	}
}

func TestPeriodicDispatcherEvent(t *testing.T) {
	d := testPeriodicDispatcher()

	period := 100 * time.Millisecond
	if err := d.Add(job("a"), period); err != nil {
		t.Fatal(err)
	}

	dur0 := waitForEvent(d, "a", t)
	validPeriod(dur0, period, t)

	dur1 := waitForEvent(d, "a", t)
	validPeriod(dur1, period, t)

	d.SetEnabled(false)

	select {
	case e := <-d.Events():
		t.Fatalf("event %d not expected", e)
	case <-time.After(150 * time.Millisecond):
	}
}

func TestPeriodicDispatcherMultipleEvents(t *testing.T) {
	d := testPeriodicDispatcher()

	period0 := 100 * time.Millisecond
	if err := d.Add(job("a"), period0); err != nil {
		t.Fatal(err)
	}
	period1 := 210 * time.Millisecond
	if err := d.Add(job("b"), period1); err != nil {
		t.Fatal(err)
	}

	// a
	waitForEvent(d, "a", t)
	// a
	waitForEvent(d, "a", t)
	// b
	waitForEvent(d, "b", t)
	// a
	waitForEvent(d, "a", t)
	// a
	waitForEvent(d, "a", t)
	// b
	waitForEvent(d, "b", t)
}
