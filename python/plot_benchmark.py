import os
import re

import matplotlib.pyplot as plt
import numpy as np
import pandas as pd

noop_dir = "noop-local-5"
noop_files = list(map(lambda file_name: f"{noop_dir}/{file_name}", sorted(os.listdir(noop_dir))))

pure_dir = "pure-local-5"
pure_files = list(map(lambda file_name: f"{pure_dir}/{file_name}", sorted(os.listdir(pure_dir))))

if len(noop_files) != len(pure_files):
    raise ValueError(f"noop files {len(noop_files)} vs pure files {len(pure_files)}")


def plot_benchmark(filename: str, ax=None):
    with open(filename, "r") as file:
        latencies = []
        end_times = []
        for line in file:
            if len(line) < 3:
                continue

            split = line.strip().split(',')
            start_timestamp = int(split[0])
            end_timestamp = int(split[1])
            error = split[2]
            if error == 'true':
                print('Entry with error encountered')
                continue
            latencies.append(end_timestamp - start_timestamp)
            end_times.append(int(end_timestamp / 1000 / 1000 / 1000))
            # end_times.append(datetime.datetime.fromtimestamp())  # from nanoseconds to seconds

    end_times = np.subtract(end_times, np.min(end_times))
    np.sort(end_times)
    unique, counts = np.unique(end_times, return_counts=True)
    second_counts = dict(zip(unique, counts))
    for i in range(unique[-1]):
        if i not in second_counts:
            second_counts[i] = 0

    rolling_window = np.convolve(counts[10:70], np.ones(1), 'valid') / 1
    mean_latency = np.mean(latencies)
    print(f'Mean latency: {mean_latency}')

    # df = pd.DataFrame(np.ones(len(end_times)), index=end_times)
    df = pd.DataFrame(rolling_window)
    # df = df.rolling(window="1S").sum()
    # with pd.option_context('display.max_rows', None,
    #                        'display.max_columns', None,
    #                        'display.precision', 3,
    #                        ):
    #     print(df)

    if ax is None:
        return df.plot()
    else:
        df.plot(ax=ax)


files_count = len(noop_files)
for i in range(files_count):
    noop_file = noop_files[i]
    match = re.search(".*?/([0-9]+)-.*", noop_file)
    noop_clients = int(match.group(1))

    pure_file = pure_files[i]
    match = re.search(".*?/([0-9]+)+-.*", pure_file)
    pure_clients = int(match.group(1))

    print(pure_clients)

    if noop_clients != pure_clients:
        raise ValueError(f"Noop clients {noop_clients} vs. Pure clients {pure_clients}")

    ax = plot_benchmark(noop_file)
    plot_benchmark(pure_file, ax)
    ax.legend(['Noop', 'Pure'])
    ax.set_xlabel('Seconds')
    ax.set_ylabel('Throughput [req/s]')
    ax.set_title(f'Throughput for {noop_clients} clients')
    plt.show()
    plt.clf()
