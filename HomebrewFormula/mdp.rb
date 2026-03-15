class Mdpaste < Formula
  desc "Paste clipboard image as Markdown link"
  homepage "https://github.com/daaa1k/mdp"
  version "0.1.0"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/daaa1k/mdp/releases/download/v#{version}/mdp-macos-aarch64"
      sha256 "" # fill in after first release

      def install
        bin.install "mdp-macos-aarch64" => "mdp"
      end
    else
      url "https://github.com/daaa1k/mdp/releases/download/v#{version}/mdp-macos-x86_64"
      sha256 "" # fill in after first release

      def install
        bin.install "mdp-macos-x86_64" => "mdp"
      end
    end
  end

  on_linux do
    url "https://github.com/daaa1k/mdp/releases/download/v#{version}/mdp-linux-x86_64"
    sha256 "" # fill in after first release

    def install
      bin.install "mdp-linux-x86_64" => "mdp"
    end
  end

  test do
    system "#{bin}/mdp", "--help"
  end
end
