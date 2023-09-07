class Counter:
    def __init__(self, base):
        self.i = base

    def inc(self):
        self.i += 1

    def less_than(self, j):
        return self.i < j